// main.go — Cartoon Scheduler с веб-интерфейсом и правильными путями
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/inputs"
)

const sourceName = "VideoSource"
const obsPassword = "123456"

var currentStatus = "Ожидание запуска..."

type Cartoon struct {
	Title    string    `json:"title"`
	Episodes []Episode `json:"episodes"`
}
type Episode struct {
	Filename    string `json:"filename"`
	DurationSec int    `json:"duration_sec"`
}
type Slot struct {
	CartoonID string `json:"cartoon_id"`
	Start     string `json:"start"`
	End       string `json:"end"`
}
type Schedule struct {
	PeriodName string `json:"period_name"`
	Slots      []Slot `json:"slots"`
}
type CartoonState struct {
	LastPlayedEpisodeIndex int       `json:"last_played_episode_index"`
	LastPlayedAt           time.Time `json:"last_played_at"`
}
type GlobalState map[string]CartoonState

func strptr(s string) *string { return &s }
func boolptr(b bool) *bool    { return &b }

// Возвращает директорию, где лежит исполняемый файл
func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal("❌ Не удалось определить путь к программе:", err)
	}
	dir, err := filepath.Abs(filepath.Dir(exe))
	if err != nil {
		log.Fatal("❌ Не удалось получить абсолютный путь:", err)
	}
	return dir
}

// Загружает JSON из папки программы
func loadJSONFromProgramDir(filename string, v interface{}) {
	exeDir := getExecutableDir()
	path := filepath.Join(exeDir, filename)
	fmt.Printf("📂 Читаю: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("❌ Файл не найден: %s\nПоложите его в ту же папку, где лежит cartoon-scheduler.exe", path)
	}
	if err := json.Unmarshal(data, v); err != nil {
		log.Fatalf("❌ Ошибка в файле %s: %v", path, err)
	}
}

// Сохраняет state.json в папку программы
func saveStateToProgramDir(state GlobalState) {
	exeDir := getExecutableDir()
	path := filepath.Join(exeDir, "state.json")
	data, _ := json.MarshalIndent(state, "", "  ")
	err := os.WriteFile(path, data, 0644)
	if err != nil {
		log.Printf("⚠️ Не удалось сохранить state.json: %v", err)
	}
}

func parseClock(t string) time.Time {
	now := time.Now()
	tm, _ := time.Parse("15:04", t)
	return time.Date(now.Year(), now.Month(), now.Day(), tm.Hour(), tm.Minute(), 0, 0, now.Location())
}

func getCurrentSchedule() string {
	hour := time.Now().Hour()
	if hour >= 6 && hour < 11 {
		return "morning"
	} else if hour >= 11 && hour < 18 {
		return "day"
	}
	return "evening"
}

func findActiveSlot(sched *Schedule) *Slot {
	now := time.Now()
	for _, s := range sched.Slots {
		start := parseClock(s.Start)
		end := parseClock(s.End)
		if !now.Before(start) && now.Before(end) {
			return &s
		}
	}
	return nil
}

func runMainLogic() {
	currentStatus = "Запуск планировщика..."

	// Загружаем список мультфильмов один раз
	var allCartoons map[string]Cartoon
	loadJSONFromProgramDir("cartoons.json", &allCartoons)

	// Загружаем state
	state := make(GlobalState)
	exeDir := getExecutableDir()
	if data, err := os.ReadFile(filepath.Join(exeDir, "state.json")); err == nil {
		json.Unmarshal(data, &state)
	}

	// Подключаемся к OBS
	client, err := goobs.New("localhost:4455", goobs.WithPassword(obsPassword))
	if err != nil {
		currentStatus = "❌ Не подключился к OBS: " + err.Error()
		return
	}
	defer client.Disconnect()

	// Бесконечный цикл: каждый проход = один эпизод
	for {
		// 🔍 1. Определяем, какой мультфильм должен идти СЕЙЧАС
		// period := getCurrentSchedule()
		var currentSchedule Schedule
		loadJSONFromProgramDir("schedule_day.json", &currentSchedule)

		activeSlot := findActiveSlot(&currentSchedule)
		if activeSlot == nil {
			currentStatus = "🔚 Нет активного слота. Жду..."
			time.Sleep(30 * time.Second)
			continue
		}

		cartoonID := activeSlot.CartoonID
		cartoon, ok := allCartoons[cartoonID]
		if !ok {
			currentStatus = "❌ Мультфильм не найден: " + cartoonID
			time.Sleep(30 * time.Second)
			continue
		}

		// 🎞️ 2. Выбираем следующий эпизод из этого мультфильма
		nextIndex := 0
		if s, exists := state[cartoonID]; exists {
			nextIndex = (s.LastPlayedEpisodeIndex + 1) % len(cartoon.Episodes)
		}

		ep := cartoon.Episodes[nextIndex]
		absPath := filepath.Join(getExecutableDir(), "cartoons", cartoonID, ep.Filename)

		// ▶️ 3. Запускаем эпизод в OBS
		currentStatus = fmt.Sprintf("▶️ %s: %s", cartoon.Title, ep.Filename)
		fmt.Println(currentStatus)

		_, err := client.Inputs.SetInputSettings(&inputs.SetInputSettingsParams{
			InputName:     strptr(sourceName),
			InputSettings: map[string]interface{}{"local_file": absPath},
			Overlay:       boolptr(false),
		})
		if err != nil {
			log.Printf("OBS: %v", err)
		}

		// 💾 Сохраняем state
		state[cartoonID] = CartoonState{
			LastPlayedEpisodeIndex: nextIndex,
			LastPlayedAt:           time.Now(),
		}
		saveStateToProgramDir(state)

		// ⏳ 4. Ждём РОВНО столько, сколько длится эпизод — без прерываний
		duration := time.Duration(ep.DurationSec) * time.Second
		if duration <= 0 {
			duration = 20 * time.Minute
		}
		time.Sleep(duration)

		// → После окончания — цикл повторяется: снова проверяем "что сейчас по расписанию?"
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		period := getCurrentSchedule()
		var sched Schedule
		exeDir := getExecutableDir()
		schedulePath := filepath.Join(exeDir, "schedule_"+period+".json")
		if data, err := os.ReadFile(schedulePath); err == nil {
			json.Unmarshal(data, &sched)
		}
		active := findActiveSlot(&sched)
		activeStr := "нет активного слота"
		if active != nil {
			activeStr = active.CartoonID + " (" + active.Start + "–" + active.End + ")"
		}

		html := `
		<!DOCTYPE html>
		<html>
		<head><title>Cartoon Player</title>
		<style>
			body { font-family: Arial, sans-serif; max-width: 600px; margin: 40px auto; }
			button { padding: 10px 20px; font-size: 16px; margin: 10px 5px; }
			.status { background: #f0f0f0; padding: 15px; margin: 20px 0; border-radius: 5px; }
		</style>
		</head>
		<body>
		<h1>📺 Cartoon Player</h1>
		<p><b>Текущий период:</b> %s</p>
		<p><b>Активный слот:</b> %s</p>

		<div class="status">%s</div>

		<form method="POST" action="/start"><button type="submit">▶️ Запустить расписание</button></form>
		<form method="POST" action="/scan"><button type="submit">🔍 Сканировать мультфильмы</button></form>
		<form method="POST" action="/stop"><button type="submit" style="background:#f44336;color:white;">⏹️ Остановить</button></form>
		</body>
		</html>
		`
		fmt.Fprintf(w, html, period, activeStr, currentStatus)
	})

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		go runMainLogic()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		exeDir := getExecutableDir()
		cmd := exec.Command(filepath.Join(exeDir, "cartoon-scanner"))
		cmd.Dir = exeDir // ← Запускать из своей папки!
		err := cmd.Run()
		if err != nil {
			currentStatus = "❌ Ошибка при сканировании: " + err.Error()
		} else {
			currentStatus = "✅ Файл cartoons.json обновлён!"
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	fmt.Println("✅ Запускаю веб-интерфейс на http://localhost:8080")
	fmt.Println("   Не закрывай это окно! Открой браузер → перейди по ссылке.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
