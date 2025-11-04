package main

import (
	"strings"
)

type Player struct {
	Room      *Room
	Inventory map[string]bool
	Wearing   map[string]bool
}

type World struct {
	Rooms      map[string]*Room
	Corridor   *Room
	DoorOpened bool
}

type Room struct {
	Name    string
	Exits   []string
	OnEnter func(r *Room, p *Player, w *World) string
	OnLook  func(r *Room, p *Player, w *World) string
	Places  map[string][]string
	OnApply map[string]func(target string, p *Player, w *World) (bool, string)
}

var (
	world  *World
	player *Player
)

func main() {

}

func initGame() {
	world = &World{
		Rooms:      map[string]*Room{},
		DoorOpened: false,
	}
	player = &Player{
		Inventory: map[string]bool{},
		Wearing:   map[string]bool{},
	}

	kitchen := &Room{
		Name:  "кухня",
		Exits: []string{"коридор"},
		OnEnter: func(r *Room, p *Player, w *World) string {
			return "кухня, ничего интересного. " + exitsLine(r.Exits)
		},
		OnLook: func(r *Room, p *Player, w *World) string {
			need := "надо собрать рюкзак и идти в универ"
			if p.Wearing["рюкзак"] {
				need = "надо идти в универ"
			}
			return "ты находишься на кухне, на столе: чай, " + need + ". " + exitsLine(r.Exits)
		},
	}

	corridor := &Room{
		Name:  "коридор",
		Exits: []string{"кухня", "комната", "улица"},
		OnEnter: func(r *Room, p *Player, w *World) string {
			return "ничего интересного. " + exitsLine(r.Exits)
		},
		OnLook: func(r *Room, p *Player, w *World) string {
			return "ничего интересного. " + exitsLine(r.Exits)
		},
		OnApply: map[string]func(target string, p *Player, w *World) (bool, string){
			"ключи": func(target string, p *Player, w *World) (bool, string) {
				if target == "дверь" {
					if w.DoorOpened {
						w.DoorOpened = false
						return true, "дверь закрыта"
					}
					w.DoorOpened = true
					return true, "дверь открыта"
				}
				return false, ""
			},
		},
	}

	world.Corridor = corridor

	room := &Room{
		Name:  "комната",
		Exits: []string{"коридор"},
		Places: map[string][]string{
			"на столе": {"ключи", "конспекты"},
			"на стуле": {"рюкзак"},
		},
		OnEnter: func(r *Room, p *Player, w *World) string {
			return "ты в своей комнате. " + exitsLine(r.Exits)
		},
		OnLook: func(r *Room, p *Player, w *World) string {
			parts := []string{}
			if items := r.Places["на столе"]; len(items) > 0 {
				parts = append(parts, "на столе: "+joinComma(items))
			}
			if items := r.Places["на стуле"]; len(items) > 0 {
				parts = append(parts, "на стуле: "+joinComma(items))
			}
			if len(parts) == 0 {
				return "пустая комната. " + exitsLine(r.Exits)
			}
			return strings.Join(parts, ", ") + ". " + exitsLine(r.Exits)
		},
	}

	street := &Room{
		Name:  "улица",
		Exits: []string{"домой"},
		OnEnter: func(r *Room, p *Player, w *World) string {
			return "на улице весна. " + exitsLine(r.Exits)
		},
		OnLook: func(r *Room, p *Player, w *World) string {
			return "на улице весна. " + exitsLine(r.Exits)
		},
	}

	world.Rooms["кухня"] = kitchen
	world.Rooms["коридор"] = corridor
	world.Rooms["комната"] = room
	world.Rooms["улица"] = street

	player.Room = kitchen
}

func handleCommand(command string) string {
	cmd, a1, a2 := parse(command)

	switch cmd {
	case "осмотреться":
		return player.Room.OnLook(player.Room, player, world)
	case "идти":
		if a1 == "" {
			return "неизвестная команда"
		}
		if !hasExit(player.Room, a1) {
			return "нет пути в " + a1
		}
		if a1 == "улица" && !world.DoorOpened {
			return "дверь закрыта"
		}
		dest := a1
		if a1 == "домой" {
			dest = "коридор"
		}

		next := world.Rooms[dest]
		if next == nil {
			return "нет пути в " + a1
		}
		player.Room = next
		return next.OnEnter(next, player, world)

	case "взять":
		if a1 == "" {
			return "неизвестная команда"
		}
		if !player.Wearing["рюкзак"] {
			return "некуда класть"
		}
		if removeItemFromPlaces(player.Room.Places, a1) {
			player.Inventory[a1] = true
			return "предмет добавлен в инвентарь: " + a1
		}
		return "нет такого"

	case "надеть":
		if a1 == "" {
			return "неизвестная команда"
		}
		if a1 == "рюкзак" && removeItemFromPlaces(player.Room.Places, "рюкзак") {
			player.Wearing["рюкзак"] = true
			return "вы надели: рюкзак"
		}
		return "нет такого"

	case "применить":
		if a1 == "" || a2 == "" {
			return "неизвестная команда"
		}
		if !player.Inventory[a1] {
			return "нет предмета в инвентаре - " + a1
		}
		if player.Room.OnApply != nil {
			if fn, ok := player.Room.OnApply[a1]; ok {
				if handled, out := fn(a2, player, world); handled {
					return out
				}
			}
		}
		return "не к чему применить"
	}

	return "неизвестная команда"
}

func parse(s string) (cmd, a1, a2 string) {
	parts := strings.SplitN(strings.TrimSpace(s), " ", 3)
	switch len(parts) {
	case 0:
		return "", "", ""
	case 1:
		return parts[0], "", ""
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], parts[2]
	}
}

func exitsLine(exits []string) string {
	return "можно пройти - " + strings.Join(exits, ", ")
}

func joinComma(items []string) string {
	return strings.Join(items, ", ")
}

func hasExit(r *Room, dest string) bool {
	for _, e := range r.Exits {
		if e == dest {
			return true
		}
	}
	return false
}

func removeItemFromPlaces(places map[string][]string, item string) bool {
	if places == nil {
		return false
	}
	for place, arr := range places {
		idx := -1
		for i, it := range arr {
			if it == item {
				idx = i
				break
			}
		}
		if idx >= 0 {
			arr = append(arr[:idx], arr[idx+1:]...)
			if len(arr) == 0 {
				places[place] = []string{}
			} else {
				places[place] = arr
			}
			return true
		}
	}
	return false
}
