package main

// сюда писать код

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	tgbotapi "github.com/skinass/telegram-bot-api/v5"
)

var (
	// @BotFather в телеграме даст вам токен. Если захотите потыкать своего бота через телегу - используйте именно его
	BotToken = "7709381978:AAHE9Uc9r2UXIABajno9fH5vSEqW-H9Jbv8"

	// Урл, в который будет стучаться телега при получении сообщения от пользователя.
	// Может быть как айпишником личной виртуалки, так и просто выдан сервисом для деплоя
	WebhookURL = "https://87ea7309a7f3fc625d92785b154ffe36.serveo.net"
	TaskID     atomic.Int64
)

type taskRepo interface {
	UnassignTask(ctx context.Context, userID, taskID int64) (*Task, error)
	GetTasks(ctx context.Context) []*Task
	GetTaskByID(ctx context.Context, taskID int64) (*Task, error)
	GetTasksByUserID(ctx context.Context, userID int64) ([]*Task, error)
	DeleteTaskByID(ctx context.Context, taskID int64)
	GetTasksByAsigneeID(ctx context.Context, asigneeID int64) ([]*Task, error)
	AddTask(ctx context.Context, task *Task) error
	AssignTask(ctx context.Context, userID int64, userName string, taskID int64) (*Task, error)
}

type Task struct {
	TaskID        int64
	OwnerID       int64
	OwnerName     string
	AssigneeID    int64
	AsssigneeName string
	TaskInfo      string
}

type TaskDB struct {
	mp map[int64]*Task
	mu sync.RWMutex
}

func NewTaskDB() taskRepo {
	return &TaskDB{
		mp: make(map[int64]*Task),
		mu: sync.RWMutex{},
	}
}

func (tdb *TaskDB) UnassignTask(ctx context.Context, userID, taskID int64) (*Task, error) {
	tdb.mu.Lock()
	defer tdb.mu.Unlock()

	if task, ok := tdb.mp[taskID]; !ok {
		return nil, fmt.Errorf("cant find element with such id")
	} else {
		task.AssigneeID = 0
		task.AsssigneeName = ""

		return task, nil
	}
}

func (tdb *TaskDB) GetTasks(ctx context.Context) []*Task {
	tdb.mu.RLock()
	defer tdb.mu.RUnlock()

	var tasks []*Task

	for _, value := range tdb.mp {
		tasks = append(tasks, value)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].TaskID < tasks[j].TaskID })

	return tasks
}

func (tdb *TaskDB) GetTaskByID(ctx context.Context, taskID int64) (*Task, error) {
	tdb.mu.RLock()
	defer tdb.mu.RUnlock()

	task, ok := tdb.mp[taskID]
	if !ok {
		return nil, fmt.Errorf("cant find element with such id")
	} else {
		return task, nil
	}
}

func (tdb *TaskDB) GetTasksByUserID(ctx context.Context, userID int64) ([]*Task, error) {
	tdb.mu.RLock()
	defer tdb.mu.RUnlock()

	var tasks []*Task

	for _, value := range tdb.mp {
		if value.OwnerID == userID {
			tasks = append(tasks, value)
		}
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].TaskID < tasks[j].TaskID })

	return tasks, nil
}

func (tdb *TaskDB) DeleteTaskByID(ctx context.Context, taskID int64) {
	tdb.mu.Lock()
	defer tdb.mu.Unlock()

	delete(tdb.mp, taskID)
}

func (tdb *TaskDB) GetTasksByAsigneeID(ctx context.Context, asigneeID int64) ([]*Task, error) {
	tdb.mu.RLock()
	defer tdb.mu.RUnlock()

	var tasks []*Task

	for _, value := range tdb.mp {
		if value.AssigneeID == asigneeID {
			tasks = append(tasks, value)
		}
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].TaskID < tasks[j].TaskID })

	return tasks, nil
}

func (tdb *TaskDB) AddTask(ctx context.Context, task *Task) error {
	tdb.mu.Lock()
	defer tdb.mu.Unlock()

	tdb.mp[task.TaskID] = task

	return nil
}

func (tdb *TaskDB) AssignTask(ctx context.Context, userID int64, userName string, taskID int64) (*Task, error) {
	tdb.mu.Lock()
	defer tdb.mu.Unlock()

	if task, ok := tdb.mp[taskID]; ok {
		task.AssigneeID = userID
		task.AsssigneeName = userName

		return task, nil
	} else {
		return nil, fmt.Errorf("cant change assignee")
	}
}

type Usecase struct {
	db taskRepo
}

func NewUsecase(db taskRepo) *Usecase {
	return &Usecase{
		db: db,
	}
}

func (uc *Usecase) CreateTask(ctx context.Context, update tgbotapi.Update) string {
	txt, _ := strings.CutPrefix(update.Message.Text, "/new ")
	task := &Task{
		TaskID:    TaskID.Add(1),
		OwnerID:   update.Message.Chat.ID,
		OwnerName: update.Message.From.UserName,
		TaskInfo:  txt,
	}
	uc.db.AddTask(ctx, task)
	return fmt.Sprintf("Задача \"%s\" создана, id=%d", task.TaskInfo, task.TaskID)
}

func (uc *Usecase) AssignTask(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	txt, _ := strings.CutPrefix(update.Message.Text, "/assign_")
	taskNum, err := strconv.Atoi(txt)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "wrong task ID"))
		return
	}

	tBefore, err := uc.db.GetTaskByID(ctx, int64(taskNum))
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, error happend"))
		return
	}
	prevAssigneeID := tBefore.AssigneeID
	ownerID := tBefore.OwnerID

	task, err := uc.db.AssignTask(ctx, update.Message.From.ID, update.Message.From.UserName, int64(taskNum))
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, error happend"))
		return
	}

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Задача \"%s\" назначена на вас", task.TaskInfo)))

	if prevAssigneeID != 0 && prevAssigneeID != update.Message.Chat.ID {
		bot.Send(tgbotapi.NewMessage(prevAssigneeID, fmt.Sprintf("Задача \"%s\" назначена на @%s", task.TaskInfo, update.Message.From.UserName)))
		return
	}

	if ownerID != update.Message.Chat.ID {
		bot.Send(tgbotapi.NewMessage(ownerID, fmt.Sprintf("Задача \"%s\" назначена на @%s", task.TaskInfo, update.Message.From.UserName)))
	}
}

func (uc *Usecase) UnassignTask(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	txt, _ := strings.CutPrefix(update.Message.Text, "/unassign_")
	taskNum, err := strconv.Atoi(txt)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "wrong task ID"))
		return
	}

	task, err := uc.db.GetTaskByID(ctx, int64(taskNum))
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "wrong task ID"))
		return
	}

	if task.AssigneeID == update.Message.Chat.ID {
		taskRes, err := uc.db.UnassignTask(ctx, update.Message.Chat.ID, int64(taskNum))
		if err != nil {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, error happend"))
		}

		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Принято"))
		bot.Send(tgbotapi.NewMessage(taskRes.OwnerID, fmt.Sprintf("Задача \"%s\" осталась без исполнителя", taskRes.TaskInfo)))
		return
	}

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Задача не на вас"))
}

func (uc *Usecase) CompleteTask(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	txt, _ := strings.CutPrefix(update.Message.Text, "/resolve_")
	taskNum, err := strconv.Atoi(txt)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "wrong task ID"))
		return
	}

	task, err := uc.db.GetTaskByID(ctx, int64(taskNum))
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, error happend"))
		return
	}

	taskInfo := task.TaskInfo
	taskOwnerID := task.OwnerID

	uc.db.DeleteTaskByID(ctx, int64(taskNum))

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Задача \"%s\" выполнена", task.TaskInfo)))
	bot.Send(tgbotapi.NewMessage(taskOwnerID, fmt.Sprintf("Задача \"%s\" выполнена @%s", taskInfo, update.Message.From.UserName)))
}

func (uc *Usecase) ListMyTask(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	tasks, err := uc.db.GetTasksByAsigneeID(ctx, update.Message.Chat.ID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, error happend"))
		return
	}

	if len(tasks) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Пусто"))
		return
	}

	var b strings.Builder
	for i, t := range tasks {
		fmt.Fprintf(&b, "%d. %s by @%s\n/unassign_%d /resolve_%d",
			t.TaskID, t.TaskInfo, t.OwnerName, t.TaskID, t.TaskID)
		if i != len(tasks)-1 {
			b.WriteString("\n")
		}
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, b.String()))
}

func (uc *Usecase) ListOwnTasks(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	tasks, err := uc.db.GetTasksByUserID(ctx, update.Message.Chat.ID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sorry, error happend"))
		return
	}

	if len(tasks) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Пусто"))
		return
	}

	var b strings.Builder
	for i, t := range tasks {
		fmt.Fprintf(&b, "%d. %s by @%s\n/assign_%d",
			t.TaskID, t.TaskInfo, t.OwnerName, t.TaskID)
		if i != len(tasks)-1 {
			b.WriteString("\n")
		}
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, b.String()))
}

func (uc *Usecase) ListAllTasks(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	tasks := uc.db.GetTasks(ctx)

	if len(tasks) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Нет задач"))
		return
	}

	var tasksStr strings.Builder

	for it := 0; it < len(tasks); it++ {
		switch tasks[it].AssigneeID {
		case update.Message.From.ID:
			tasksStr.WriteString(fmt.Sprintf("%d. %s by @%s\nassignee: я\n/unassign_%d /resolve_%d", tasks[it].TaskID, tasks[it].TaskInfo, tasks[it].OwnerName, tasks[it].TaskID, tasks[it].TaskID))
		case 0:
			tasksStr.WriteString(fmt.Sprintf("%d. %s by @%s\n/assign_%d", tasks[it].TaskID, tasks[it].TaskInfo, tasks[it].OwnerName, tasks[it].TaskID))
		default:
			tasksStr.WriteString(fmt.Sprintf("%d. %s by @%s\nassignee: @%s", tasks[it].TaskID, tasks[it].TaskInfo, tasks[it].OwnerName, tasks[it].AsssigneeName))
		}
		if it+1 < len(tasks) {
			tasksStr.WriteString("\n\n")
		}
	}

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, tasksStr.String()))
}

func startTaskBot(ctx context.Context) error {
	// Сюда пишите ваш код
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatalf("NewBotAPI failed: %s", err)
	}

	bot.Debug = true
	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

	db := NewTaskDB()
	uc := NewUsecase(db)

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Fatalf("NewWebhook failed: %s", err)
	}

	if _, err := bot.Request(wh); err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}

	updates := bot.ListenForWebhook("/")

	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":8081", nil))
	}()
	fmt.Println("start listen")

	for update := range updates {
		log.Printf("upd: %#v\n", update)

		msg := update.Message.Text

		if strings.HasPrefix(msg, "/new ") {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, uc.CreateTask(ctx, update)))
		} else if strings.HasPrefix(msg, "/assign_") {
			uc.AssignTask(ctx, bot, update)
		} else if strings.HasPrefix(msg, "/unassign_") {
			uc.UnassignTask(ctx, bot, update)
		} else if strings.HasPrefix(msg, "/resolve_") {
			uc.CompleteTask(ctx, bot, update)
		} else if strings.HasPrefix(msg, "/my") {
			uc.ListMyTask(ctx, bot, update)
		} else if strings.HasPrefix(msg, "/owner") {
			uc.ListOwnTasks(ctx, bot, update)
		} else if strings.HasPrefix(msg, "/tasks") {
			uc.ListAllTasks(ctx, bot, update)
		} else {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Unsupported command"))
		}
	}

	return nil
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
