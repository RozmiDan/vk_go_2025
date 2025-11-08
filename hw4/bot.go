package main

// сюда писать код

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	getTasks(ctx context.Context) []*Task
	addTask(ctx context.Context, task *Task) error
	assignUser(ctx context.Context, userID, taskID string) error
	unassignUser(ctx context.Context, userID, taskID string) error
	completeTask(ctx context.Context, taskID string) error
	getUserTasks(ctx context.Context, userID string) error
	getUserOwnTasks(ctx context.Context, userID string) error
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
	mp map[string][]*Task
	mu sync.RWMutex
}

func NewTaskDB() taskRepo {
	return &TaskDB{
		mp: make(map[string][]*Task),
		mu: sync.RWMutex{},
	}
}

func (tdb *TaskDB) getTasks(ctx context.Context) []*Task {
	return nil
}

func (tdb *TaskDB) addTask(ctx context.Context, task *Task) error {
	log.Println(*task)
	return nil
}

func (tdb *TaskDB) assignUser(ctx context.Context, userID, taskID string) error {
	return nil
}

func (tdb *TaskDB) unassignUser(ctx context.Context, userID, taskID string) error {
	return nil
}

func (tdb *TaskDB) completeTask(ctx context.Context, taskID string) error {
	return nil
}

func (tdb *TaskDB) getUserTasks(ctx context.Context, userID string) error {
	return nil
}

func (tdb *TaskDB) getUserOwnTasks(ctx context.Context, userID string) error {
	return nil
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

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Fatalf("NewWebhook failed: %s", err)
	}

	if _, err := bot.Request(wh); err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}

	updates := bot.ListenForWebhook("/")

	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":80", nil))
	}()
	fmt.Println("start listen")

	for update := range updates {
		log.Printf("upd: %#v\n", update)

		// usrID := update.Message.From.ID
		// usrTag := update.Message.From.UserName
		msg := update.Message.Text

		if strings.HasPrefix(msg, "/new ") {
			txt, _ := strings.CutPrefix(msg, "/new ")
			db.addTask(ctx, &Task{
				TaskID:    TaskID.Add(1),
				OwnerID:   update.Message.From.ID,
				OwnerName: update.Message.From.UserName,
				TaskInfo:  txt,
			})
		} else if strings.HasPrefix(msg, "/assign_") {

		} else if strings.HasPrefix(msg, "/unassign_") {

		} else if strings.HasPrefix(msg, "/resolve_") {

		} else if strings.HasPrefix(msg, "/my") {

		} else if strings.HasPrefix(msg, "/owner") {

		} else {

		}

		// bot.Send(msg)
		// continue

		// bot.Send(tgbotapi.NewMessage(
		// 	update.Message.Chat.ID,
		// 	"sorry, error happend",
		// ))

		// bot.Send(tgbotapi.NewMessage(
		// 	update.Message.Chat.ID,
		// 	item.URL+"\n"+item.Title,
		// ))
	}

	return nil
}

// * `/tasks` - выводит список всех активных задач
// * `/new XXX YYY ZZZ` - создаёт новую задачу
// * `/assign_$ID` - делает пользователя исполнителем задачи
// * `/unassign_$ID` - снимает задачу с текущего исполнителя
// * `/resolve_$ID` - выполняет задачу, удаляет её из списка
// * `/my` - показывает задачи, которые назначены на меня
// * `/owner` - показывает задачи, которые были созданы мной

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
