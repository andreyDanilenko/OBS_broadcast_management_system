package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Episode struct {
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
}

type Cartoon struct {
	Title         string    `json:"title"`
	TotalEpisodes int       `json:"total_episodes"`
	Episodes      []Episode `json:"episodes"`
}

type Slot struct {
	CartoonID string `json:"cartoon_id"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type Schedule struct {
	PeriodName string `json:"period_name"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Slots      []Slot `json:"slots"`
}

// findDataJSON ищет файл data.json в разных возможных местах
func findDataJSON() string {
	// 1. Пробуем найти рядом с исполняемым файлом
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		dataPath := filepath.Join(exeDir, "cartoons.json")
		if _, err := os.Stat(dataPath); err == nil {
			fmt.Printf("🔍 Найден data.json рядом с программой: %s\n", dataPath)
			return dataPath
		}
	}

	// 2. Пробуем в текущей рабочей директории
	wd, err := os.Getwd()
	if err == nil {
		dataPath := filepath.Join(wd, "cartoons.json")
		if _, err := os.Stat(dataPath); err == nil {
			fmt.Printf("🔍 Найден data.json в текущей директории: %s\n", dataPath)
			return dataPath
		}
	}

	// 3. Пробуем в родительской директории
	if err == nil {
		parentDir := filepath.Dir(wd)
		dataPath := filepath.Join(parentDir, "cartoons.json")
		if _, err := os.Stat(dataPath); err == nil {
			fmt.Printf("🔍 Найден data.json в родительской директории: %s\n", dataPath)
			return dataPath
		}
	}

	// 4. Если ничего не нашли, возвращаем путь рядом с исполняемым файлом
	exe, _ = os.Executable()
	exeDir := filepath.Dir(exe)
	defaultPath := filepath.Join(exeDir, "cartoons.json")
	fmt.Printf("⚠ data.json не найден, будет использован путь: %s\n", defaultPath)
	return defaultPath
}

func main() {
	// Находим data.json
	dataPath := findDataJSON()

	// Читаем JSON файл
	file, err := os.ReadFile(dataPath)
	if err != nil {
		log.Fatalf("❌ Ошибка чтения файла %s: %v", dataPath, err)
	}

	// Используем map[string]Cartoon для хранения всех мультфильмов
	var data map[string]Cartoon
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Fatal("❌ Ошибка парсинга JSON:", err)
	}

	// Создаем карту мультфильмов
	cartoons := make(map[string]Cartoon)
	availableCartoons := make([]string, 0, len(data))

	for cartoonID, cartoon := range data {
		cartoons[cartoonID] = cartoon
		availableCartoons = append(availableCartoons, cartoonID)
	}

	// Инициализируем расписания
	schedules := map[string]*Schedule{
		"morning": {
			PeriodName: "morning",
			StartTime:  "06:00",
			EndTime:    "12:00",
			Slots:      []Slot{},
		},
		"day": {
			PeriodName: "day",
			StartTime:  "12:00",
			EndTime:    "18:00",
			Slots:      []Slot{},
		},
		"evening": {
			PeriodName: "evening",
			StartTime:  "18:00",
			EndTime:    "00:00",
			Slots:      []Slot{},
		},
	}

	currentTime := parseTime("06:00") // Начинаем с утра
	currentPeriod := "morning"

	fmt.Println("🎬 Программа планирования мультфильмов")
	fmt.Println("========================================")
	fmt.Println("Введите мультфильмы в формате 'название количество_серий'")
	fmt.Println("Для завершения введите 'exit'")
	fmt.Printf("📺 Доступные мультфильмы: %s\n", strings.Join(availableCartoons, ", "))
	fmt.Println("----------------------------------------")

	// Создаем reader для ввода
	reader := bufio.NewReader(os.Stdin)

	// Переменные для отслеживания текущего мультфильма
	var currentCartoon string
	var slotStartTime time.Time
	var totalDuration time.Duration

	// Обработка ввода
	for {
		fmt.Print("🎯 > ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "exit" {
			// Завершаем последний слот перед выходом
			if currentCartoon != "" {
				addSlotToSchedule(schedules[currentPeriod], currentCartoon, slotStartTime, totalDuration)
				fmt.Printf("✅ Завершен слот: %s (%s - %s)\n",
					currentCartoon, formatTime(slotStartTime), formatTime(slotStartTime.Add(totalDuration)))
			}
			break
		}

		parts := strings.Fields(input)
		if len(parts) < 2 {
			fmt.Println("❌ Ошибка: введите название и количество серий через пробел")
			continue
		}

		cartoonName := strings.ToLower(parts[0])
		episodesCount, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Println("❌ Ошибка: количество серий должно быть числом")
			continue
		}

		cartoon, exists := cartoons[cartoonName]
		if !exists {
			fmt.Printf("❌ Ошибка: мультфильм '%s' не найден\n", cartoonName)
			fmt.Printf("📋 Доступные: %s\n", strings.Join(availableCartoons, ", "))
			continue
		}

		if episodesCount > len(cartoon.Episodes) {
			fmt.Printf("⚠ Предупреждение: запрошено %d серий, но доступно только %d\n",
				episodesCount, len(cartoon.Episodes))
			episodesCount = len(cartoon.Episodes)
		}

		// Обрабатываем эпизоды
		for i := 0; i < episodesCount; i++ {
			if i >= len(cartoon.Episodes) {
				break
			}

			episode := cartoon.Episodes[i]
			duration := time.Duration(episode.Duration) * time.Second

			// Проверяем смену мультфильма
			if cartoonName != currentCartoon {
				// Завершаем предыдущий слот
				if currentCartoon != "" {
					addSlotToSchedule(schedules[currentPeriod], currentCartoon, slotStartTime, totalDuration)
					fmt.Printf("✅ Завершен слот: %s (%s - %s)\n",
						currentCartoon, formatTime(slotStartTime), formatTime(slotStartTime.Add(totalDuration)))
					currentTime = slotStartTime.Add(totalDuration)
				}

				// Начинаем новый слот
				currentCartoon = cartoonName
				slotStartTime = currentTime
				totalDuration = 0
			}

			// Проверяем, помещается ли эпизод в текущий период
			endTime := currentTime.Add(duration)
			if !isTimeInPeriod(endTime, schedules[currentPeriod]) {
				// Завершаем текущий слот
				if currentCartoon != "" {
					addSlotToSchedule(schedules[currentPeriod], currentCartoon, slotStartTime, totalDuration)
					fmt.Printf("✅ Завершен слот: %s (%s - %s)\n",
						currentCartoon, formatTime(slotStartTime), formatTime(slotStartTime.Add(totalDuration)))
				}

				// Переходим к следующему периоду
				switch currentPeriod {
				case "morning":
					currentPeriod = "day"
					currentTime = parseTime(schedules["day"].StartTime)
					fmt.Println("🕛 Переход к дневному расписанию")
				case "day":
					currentPeriod = "evening"
					currentTime = parseTime(schedules["evening"].StartTime)
					fmt.Println("🕕 Переход к вечернему расписанию")
				case "evening":
					fmt.Println("❌ Достигнут конец дня, нельзя добавить больше серий")
					break
				}

				// Начинаем новый слот в новом периоде
				currentCartoon = cartoonName
				slotStartTime = currentTime
				totalDuration = 0
			}

			// Добавляем продолжительность эпизода
			totalDuration += duration
			currentTime = currentTime.Add(duration)

			fmt.Printf("📹 Добавлена серия: %s - %s (длительность: %s)\n",
				cartoonName, episode.Title, formatDuration(duration))
		}
	}

	// Сохраняем расписания в файлы
	saveScheduleToFile(schedules["morning"], "schedule_morning.json")
	saveScheduleToFile(schedules["day"], "schedule_day.json")
	saveScheduleToFile(schedules["evening"], "schedule_evening.json")

	fmt.Println("\n✅ Расписание сохранено в файлы:")
	fmt.Println("   - schedule_morning.json")
	fmt.Println("   - schedule_day.json")
	fmt.Println("   - schedule_evening.json")

	// Выводим статистику
	fmt.Println("\n📊 Статистика:")
	for period, schedule := range schedules {
		fmt.Printf("   %s: %d слотов\n", period, len(schedule.Slots))
	}
}

func addSlotToSchedule(schedule *Schedule, cartoonID string, start time.Time, duration time.Duration) {
	if duration == 0 {
		return
	}

	slot := Slot{
		CartoonID: cartoonID,
		Start:     formatTime(start),
		End:       formatTime(start.Add(duration)),
	}
	schedule.Slots = append(schedule.Slots, slot)
}

func parseTime(timeStr string) time.Time {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		log.Fatal("❌ Ошибка парсинга времени:", err)
	}
	return t
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}

func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d мин %d сек", minutes, seconds)
}

func isTimeInPeriod(t time.Time, schedule *Schedule) bool {
	start := parseTime(schedule.StartTime)
	end := parseTime(schedule.EndTime)

	// Если end время 00:00, считаем его как 24:00
	if end.Hour() == 0 && end.Minute() == 0 {
		end = end.Add(24 * time.Hour)
	}

	return (t.Equal(start) || t.After(start)) && (t.Before(end) || t.Equal(end))
}

func saveScheduleToFile(schedule *Schedule, filename string) {
	file, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		log.Printf("❌ Ошибка при маршалинге расписания %s: %v", schedule.PeriodName, err)
		return
	}

	err = os.WriteFile(filename, file, 0644)
	if err != nil {
		log.Printf("❌ Ошибка при записи файла %s: %v", filename, err)
		return
	}
}
