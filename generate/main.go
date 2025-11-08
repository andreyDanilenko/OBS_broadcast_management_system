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

type CartoonStats struct {
	LastPlayedEpisodeIndex int       `json:"last_played_episode_index"`
	LastPlayedAt           time.Time `json:"last_played_at"`
}

type Stats struct {
	Cartoons map[string]CartoonStats `json:"cartoons"`
}

// getExecutableDir возвращает путь к директории, где находится исполняемый файл
func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		// Если не удалось получить путь к исполняемому файлу, используем текущую директорию
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("❌ Не удалось определить текущую директорию:", err)
		}
		return wd
	}
	return filepath.Dir(exe)
}

// findDataJSON ищет файл data.json в разных возможных местах
func findDataJSON() string {
	executableDir := getExecutableDir()

	// 1. Пробуем найти рядом с исполняемым файлом
	dataPath := filepath.Join(executableDir, "cartoons.json")
	if _, err := os.Stat(dataPath); err == nil {
		fmt.Printf("🔍 Найден data.json рядом с программой: %s\n", dataPath)
		return dataPath
	}

	// 2. Пробуем в текущей рабочей директории
	wd, err := os.Getwd()
	if err == nil && wd != executableDir {
		dataPath := filepath.Join(wd, "cartoons.json")
		if _, err := os.Stat(dataPath); err == nil {
			fmt.Printf("🔍 Найден data.json в текущей директории: %s\n", dataPath)
			return dataPath
		}
	}

	// 3. Если ничего не нашли, возвращаем путь рядом с исполняемым файлом
	fmt.Printf("⚠ data.json не найден, будет использован путь: %s\n", dataPath)
	return dataPath
}

// loadExistingSchedule загружает существующее расписание если оно есть
func loadExistingSchedule() (*Schedule, time.Time) {
	executableDir := getExecutableDir()
	schedulePath := filepath.Join(executableDir, "schedule_full_day.json")

	var schedule Schedule
	var lastEndTime time.Time

	if _, err := os.Stat(schedulePath); err == nil {
		// Файл расписания существует, загружаем его
		file, err := os.ReadFile(schedulePath)
		if err != nil {
			fmt.Printf("⚠ Не удалось прочитать существующее расписание: %v\n", err)
			return &Schedule{
				PeriodName: "full_day",
				StartTime:  "06:00",
				EndTime:    "00:00",
				Slots:      []Slot{},
			}, parseTime("06:00")
		}

		err = json.Unmarshal(file, &schedule)
		if err != nil {
			fmt.Printf("⚠ Не удалось распарсить существующее расписание: %v\n", err)
			return &Schedule{
				PeriodName: "full_day",
				StartTime:  "06:00",
				EndTime:    "00:00",
				Slots:      []Slot{},
			}, parseTime("06:00")
		}

		// Определяем время окончания последнего слота
		if len(schedule.Slots) > 0 {
			lastSlot := schedule.Slots[len(schedule.Slots)-1]
			lastEndTime = parseTime(lastSlot.End)
			fmt.Printf("📅 Загружено существующее расписание, последний слот закончился в %s\n", lastSlot.End)
		} else {
			lastEndTime = parseTime("06:00")
		}

		return &schedule, lastEndTime
	}

	// Расписания нет, создаем новое
	fmt.Println("📅 Существующее расписание не найдено, создаем новое")
	return &Schedule{
		PeriodName: "full_day",
		StartTime:  "06:00",
		EndTime:    "00:00",
		Slots:      []Slot{},
	}, parseTime("06:00")
}

// loadStats загружает статистику просмотров
func loadStats() *Stats {
	executableDir := getExecutableDir()
	statsPath := filepath.Join(executableDir, "stats.json")

	var stats Stats
	stats.Cartoons = make(map[string]CartoonStats)

	if _, err := os.Stat(statsPath); err == nil {
		// Файл статистики существует, загружаем его
		file, err := os.ReadFile(statsPath)
		if err != nil {
			fmt.Printf("⚠ Не удалось прочитать статистику: %v\n", err)
			return &stats
		}

		err = json.Unmarshal(file, &stats)
		if err != nil {
			fmt.Printf("⚠ Не удалось распарсить статистику: %v\n", err)
			return &stats
		}

		fmt.Println("📊 Загружена статистика просмотров")
		for cartoonID, cartoonStats := range stats.Cartoons {
			fmt.Printf("   %s: последняя серия %d\n", cartoonID, cartoonStats.LastPlayedEpisodeIndex)
		}
	} else {
		fmt.Println("📊 Статистика не найдена, начинаем с начала")
	}

	return &stats
}

// getNextEpisodeIndex возвращает индекс следующей серии для мультфильма
func getNextEpisodeIndex(cartoonID string, stats *Stats, totalEpisodes int, sessionProgress map[string]int) int {
	// Сначала проверяем прогресс в текущей сессии
	if sessionProgress != nil {
		if lastIndex, exists := sessionProgress[cartoonID]; exists {
			nextIndex := lastIndex + 1
			if nextIndex >= totalEpisodes {
				return 0 // Начинаем сначала если дошли до конца
			}
			return nextIndex
		}
	}

	// Если в сессии нет прогресса, проверяем статистику
	if stats != nil && stats.Cartoons != nil {
		if cartoonStats, exists := stats.Cartoons[cartoonID]; exists {
			nextIndex := cartoonStats.LastPlayedEpisodeIndex + 1
			if nextIndex >= totalEpisodes {
				return 0 // Начинаем сначала если дошли до конца
			}
			return nextIndex
		}
	}

	return 0 // Если мультфильм не найден нигде, начинаем с начала
}

// updateSessionProgress обновляет прогресс в текущей сессии
func updateSessionProgress(sessionProgress map[string]int, cartoonID string, startIndex int, episodesCount int, totalEpisodes int) {
	if sessionProgress == nil {
		return
	}

	lastEpisodeIndex := (startIndex + episodesCount - 1) % totalEpisodes
	sessionProgress[cartoonID] = lastEpisodeIndex
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

func saveScheduleToFile(schedule *Schedule, filepath string) {
	file, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		log.Printf("❌ Ошибка при маршалинге расписания %s: %v", schedule.PeriodName, err)
		return
	}

	err = os.WriteFile(filepath, file, 0644)
	if err != nil {
		log.Printf("❌ Ошибка при записи файла %s: %v", filepath, err)
		return
	}
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

	// Загружаем существующее расписание и статистику
	schedule, currentTime := loadExistingSchedule()
	stats := loadStats()

	// Карта для отслеживания прогресса в текущей сессии
	sessionProgress := make(map[string]int)

	fmt.Println("🎬 Программа планирования мультфильмов")
	fmt.Println("========================================")
	fmt.Println("Введите мультфильмы в формате 'название количество_серий'")
	fmt.Println("Для завершения введите 'exit'")
	fmt.Printf("📺 Доступные мультфильмы: %s\n", strings.Join(availableCartoons, ", "))
	fmt.Printf("⏰ Начинаем с времени: %s\n", formatTime(currentTime))
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
				addSlotToSchedule(schedule, currentCartoon, slotStartTime, totalDuration)
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

		// Определяем с какой серии начинать (учитывая и статистику и текущую сессию)
		startEpisodeIndex := getNextEpisodeIndex(cartoonName, stats, len(cartoon.Episodes), sessionProgress)
		fmt.Printf("🎬 Начинаем %s с серии %d\n", cartoonName, startEpisodeIndex+1)

		// Проверяем смену мультфильма
		if cartoonName != currentCartoon {
			// Завершаем предыдущий слот
			if currentCartoon != "" {
				addSlotToSchedule(schedule, currentCartoon, slotStartTime, totalDuration)
				fmt.Printf("✅ Завершен слот: %s (%s - %s)\n",
					currentCartoon, formatTime(slotStartTime), formatTime(slotStartTime.Add(totalDuration)))
				currentTime = slotStartTime.Add(totalDuration)
			}

			// Начинаем новый слот
			currentCartoon = cartoonName
			slotStartTime = currentTime
			totalDuration = 0
		}

		// Обрабатываем эпизоды
		for i := 0; i < episodesCount; i++ {
			episodeIndex := (startEpisodeIndex + i) % len(cartoon.Episodes)
			episode := cartoon.Episodes[episodeIndex]
			duration := time.Duration(episode.Duration) * time.Second

			// Проверяем, не выходим ли за пределы дня (00:00)
			endTime := currentTime.Add(duration)
			scheduleEnd := parseTime(schedule.EndTime)
			if scheduleEnd.Hour() == 0 && scheduleEnd.Minute() == 0 {
				scheduleEnd = scheduleEnd.Add(24 * time.Hour)
			}

			if endTime.After(scheduleEnd) {
				fmt.Println("❌ Достигнут конец дня, нельзя добавить больше серий")
				break
			}

			// Добавляем продолжительность эпизода
			totalDuration += duration
			currentTime = currentTime.Add(duration)

			fmt.Printf("📹 Добавлена серия %d: %s - %s (длительность: %s)\n",
				episodeIndex+1, cartoonName, episode.Title, formatDuration(duration))
		}

		// Обновляем прогресс в текущей сессии
		updateSessionProgress(sessionProgress, cartoonName, startEpisodeIndex, episodesCount, len(cartoon.Episodes))
		fmt.Printf("📝 Прогресс %s обновлен: последняя серия %d\n",
			cartoonName, sessionProgress[cartoonName]+1)
	}

	// Сохраняем расписание в ту же директорию, где находится программа
	executableDir := getExecutableDir()
	schedulePath := filepath.Join(executableDir, "schedule_day.json")
	saveScheduleToFile(schedule, schedulePath)

	fmt.Printf("\n✅ Расписание сохранено в файл:\n")
	fmt.Printf("   - %s\n", schedulePath)

	// Выводим статистику
	fmt.Printf("\n📊 Статистика: %d слотов\n", len(schedule.Slots))

	// Выводим всё расписание
	fmt.Println("\n📅 Полное расписание:")
	for i, slot := range schedule.Slots {
		fmt.Printf("   %d. %s: %s - %s\n", i+1, slot.CartoonID, slot.Start, slot.End)
	}

	// Выводим итоговый прогресс сессии
	fmt.Println("\n🎯 Итоговый прогресс в этой сессии:")
	for cartoonID, lastIndex := range sessionProgress {
		fmt.Printf("   %s: последняя серия %d\n", cartoonID, lastIndex+1)
	}
}
