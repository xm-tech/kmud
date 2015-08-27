package session

import (
	"strings"

	"github.com/Cristofori/kmud/combat"
	"github.com/Cristofori/kmud/events"
	"github.com/Cristofori/kmud/model"
	"github.com/Cristofori/kmud/types"
	"github.com/Cristofori/kmud/utils"
)

type action struct {
	alias string
	exec  func(*Session, []string)
}

var actions = map[string]action{
	"l": aAlias("look"),
	"look": {
		exec: func(s *Session, args []string) {
			if len(args) == 0 {
				s.printRoom()
			} else if len(args) == 1 {
				arg := types.StringToDirection(args[0])

				if arg == types.DirectionNone {
					charList := model.CharactersIn(s.room.GetId())
					index := utils.BestMatch(args[0], charList.Names())

					if index == -2 {
						s.printError("Which one do you mean?")
					} else if index != -1 {
						s.printLine("Looking at: %s", charList[index].GetName())
					} else {
						itemList := model.ItemsIn(s.room)
						index = utils.BestMatch(args[0], itemList.Names())

						if index == -1 {
							s.printLine("Nothing to see")
						} else if index == -2 {
							s.printError("Which one do you mean?")
						} else {
							s.printLine("Looking at: %s", itemList[index].GetName())
						}
					}
				} else {
					if s.room.HasExit(arg) {
						loc := s.room.NextLocation(arg)
						roomToSee := model.GetRoomByLocation(loc, s.room.GetZoneId())
						if roomToSee != nil {
							area := model.GetArea(roomToSee.GetAreaId())
							s.printLine(roomToSee.ToString(model.PlayerCharactersIn(roomToSee.GetId(), nil),
								model.NpcsIn(roomToSee.GetId()), nil, area))
						} else {
							s.printLine("Nothing to see")
						}
					} else {
						s.printError("You can't look in that direction")
					}
				}
			}
		},
	},
	"a": aAlias("attack"),
	"attack": {
		exec: func(s *Session, args []string) {
			charList := model.CharactersIn(s.room.GetId())
			index := utils.BestMatch(args[0], charList.Names())

			if index == -1 {
				s.printError("Not found")
			} else if index == -2 {
				s.printError("Which one do you mean?")
			} else {
				defender := charList[index]
				if defender.GetId() == s.player.GetId() {
					s.printError("You can't attack yourself")
				} else {
					combat.StartFight(s.player, defender)
				}
			}
		},
	},
	"c": aAlias("cast"),
	"cast": {
		exec: func(s *Session, args []string) {
			var target types.Character

			if len(args) == 1 {
				target = s.player
			} else if len(args) == 2 {
				charList := model.CharactersIn(s.room.GetId())
				index := utils.BestMatch(args[0], charList.Names())

				if index == -1 {
					s.printError("Target not found")
				} else if index == -2 {
					s.printError("Which one do you mean?")
				} else {
					target = charList[index]
				}
			}

			if target != nil {
				s.printLine("Casting on", target.GetName())
			}
		},
	},
	"talk": {
		exec: func(s *Session, args []string) {
			if len(args) != 1 {
				s.printError("Usage: talk <NPC name>")
				return
			}

			npcList := model.NpcsIn(s.room.GetId())
			index := utils.BestMatch(args[0], npcList.Characters().Names())

			if index == -1 {
				s.printError("Not found")
			} else if index == -2 {
				s.printError("Which one do you mean?")
			} else {
				npc := npcList[index]
				s.printLine(npc.PrettyConversation())
			}
		},
	},
	"drop": {
		exec: func(s *Session, args []string) {
			dropUsage := func() {
				s.printError("Usage: drop <item name>")
			}

			if len(args) != 1 {
				dropUsage()
				return
			}

			characterItems := model.GetItems(s.player.GetItems())
			index := utils.BestMatch(args[0], characterItems.Names())

			if index == -1 {
				s.printError("Not found")
			} else if index == -2 {
				s.printError("Which one do you mean?")
			} else {
				item := characterItems[index]
				s.player.RemoveItem(item.GetId())
				s.room.AddItem(item.GetId())
				s.printLine("Dropped %s", item.GetName())
			}
		},
	},
	"take": aAlias("get"),
	"t":    aAlias("get"),
	"g":    aAlias("g"),
	"get": {
		exec: func(s *Session, args []string) {
			takeUsage := func() {
				s.printError("Usage: take <item name>")
			}

			if len(args) != 1 {
				takeUsage()
				return
			}

			itemsInRoom := model.GetItems(s.room.GetItems())
			index := utils.BestMatch(args[0], itemsInRoom.Names())

			if index == -2 {
				s.printError("Which one do you mean?")
			} else if index == -1 {
				s.printError("Item %s not found", args[0])
			} else {
				item := itemsInRoom[index]
				s.player.AddItem(item.GetId())
				s.room.RemoveItem(item.GetId())
				s.printLine("Picked up %s", item.GetName())
			}
		},
	},
	"i":   aAlias("inventory"),
	"inv": aAlias("inv"),
	"inventory": {
		exec: func(s *Session, args []string) {
			itemIds := s.player.GetItems()

			if len(itemIds) == 0 {
				s.printLine("You aren't carrying anything")
			} else {
				var itemNames []string
				for _, item := range model.GetItems(itemIds) {
					itemNames = append(itemNames, item.GetName())
				}
				s.printLine("You are carrying: %s", strings.Join(itemNames, ", "))
			}

			s.printLine("Cash: %v", s.player.GetCash())
		},
	},
	"help": {
		exec: func(s *Session, args []string) {
			s.printLine("HELP!")
		},
	},
	"ls": {
		exec: func(s *Session, args []string) {
			s.printLine("Where do you think you are?!")
		},
	},
	"stop": {
		exec: func(s *Session, args []string) {
			combat.StopFight(s.player)
		},
	},
	"go": {
		exec: func(s *Session, args []string) {
			if len(args) != 1 {
				s.printError("Usage: go <name>")
				return
			}

			links := s.room.GetLinks()
			linkNames := s.room.LinkNames()
			index := utils.BestMatch(args[0], linkNames)

			if index == -2 {
				s.printError("Which one do you mean?")
			} else if index == -1 {
				s.printError("Exit %s not found", args[0])
			} else {
				destId := links[linkNames[index]]
				newRoom := model.GetRoom(destId)

				model.MoveCharacterToRoom(s.player, newRoom)

				s.room = newRoom
				s.printRoom()
			}
		},
	},
	"lock": {
		exec: func(s *Session, args []string) {
			if len(args) != 1 {
				s.printError("Usage: lock <direction>")
			}
			handleLock(s, args[0], true)
		},
	},
	"unlock": {
		exec: func(s *Session, args []string) {
			if len(args) != 1 {
				s.printError("Usage: unlock <direction>")
			}
			handleLock(s, args[0], false)
		},
	},
}

func handleLock(s *Session, arg string, locked bool) {
	dir := types.StringToDirection(arg)

	if dir == types.DirectionNone {
		s.printError("Invalid direction")
	} else {
		s.room.SetLocked(dir, locked)

		events.Broadcast(events.LockEvent{
			RoomId: s.room.GetId(),
			Exit:   dir,
			Locked: locked,
		})

		// Lock on both sides
		location := s.room.NextLocation(dir)
		otherRoom := model.GetRoomByLocation(location, s.room.GetZoneId())

		if otherRoom != nil {
			otherRoom.SetLocked(dir.Opposite(), locked)

			events.Broadcast(events.LockEvent{
				RoomId: otherRoom.GetId(),
				Exit:   dir.Opposite(),
				Locked: locked,
			})
		}
	}
}

func aAlias(name string) action {
	return action{alias: name}
}
