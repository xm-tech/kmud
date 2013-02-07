package game

import (
	"fmt"
	"io"
	"kmud/database"
	"kmud/model"
	"kmud/utils"
	"labix.org/v2/mgo/bson"
	"strconv"
	"strings"
	"time"
)

type userInputMode int

const (
	CleanUserInput userInputMode = iota
	RawUserInput   userInputMode = iota
)

func toggleExitMenu(cm utils.ColorMode, room *database.Room) utils.Menu {
	onOrOff := func(direction database.ExitDirection) string {
		text := "Off"
		if room.HasExit(direction) {
			text = "On"
		}
		return utils.Colorize(cm, utils.ColorBlue, text)
	}

	menu := utils.NewMenu("Edit Exits")

	menu.AddAction("n", "[N]orth: "+onOrOff(database.DirectionNorth))
	menu.AddAction("ne", "[NE]North East: "+onOrOff(database.DirectionNorthEast))
	menu.AddAction("e", "[E]ast: "+onOrOff(database.DirectionEast))
	menu.AddAction("se", "[SE]South East: "+onOrOff(database.DirectionSouthEast))
	menu.AddAction("s", "[S]outh: "+onOrOff(database.DirectionSouth))
	menu.AddAction("sw", "[SW]South West: "+onOrOff(database.DirectionSouthWest))
	menu.AddAction("w", "[W]est: "+onOrOff(database.DirectionWest))
	menu.AddAction("nw", "[NW]North West: "+onOrOff(database.DirectionNorthWest))
	menu.AddAction("u", "[U]p: "+onOrOff(database.DirectionUp))
	menu.AddAction("d", "[D]own: "+onOrOff(database.DirectionDown))

	return menu
}

func npcMenu(room *database.Room) utils.Menu {
	npcs := model.M.NpcsIn(room.Id)

	menu := utils.NewMenu("NPCs")

	menu.AddAction("n", "[N]ew")

	for i, npc := range npcs {
		index := i + 1
		actionText := fmt.Sprintf("[%v]%v", index, npc.PrettyName())
		menu.AddActionData(index, actionText, npc.Id)
	}

	return menu
}

func specificNpcMenu(npcId bson.ObjectId) utils.Menu {
	npc := model.M.GetCharacter(npcId)
	menu := utils.NewMenu(npc.PrettyName())
	menu.AddAction("r", "[R]ename")
	menu.AddAction("d", "[D]elete")
	return menu
}

func Exec(conn io.ReadWriter, currentUser *database.User, currentChar *database.Character) {
	currentRoom := model.M.GetRoom(currentChar.GetRoomId())
	currentZone := model.M.GetZone(currentRoom.GetZoneId())

	printString := func(data string) {
		io.WriteString(conn, data)
	}

	printLineColor := func(color utils.Color, line string, a ...interface{}) {
		utils.WriteLine(conn, utils.Colorize(currentUser.GetColorMode(), color, fmt.Sprintf(line, a...)))
	}

	printLine := func(line string, a ...interface{}) {
		printLineColor(utils.ColorWhite, line, a...)
	}

	printError := func(err string, a ...interface{}) {
		printLineColor(utils.ColorRed, err, a...)
	}

	printRoom := func() {
		charList := model.M.CharactersIn(currentRoom.Id, currentChar.Id)
		npcList := model.M.NpcsIn(currentRoom.Id)
		printLine(currentRoom.ToString(database.ReadMode, currentUser.GetColorMode(),
			charList, npcList, model.M.GetItems(currentRoom.GetItemIds())))
	}

	printRoomEditor := func() {
		printLine(currentRoom.ToString(database.EditMode, currentUser.GetColorMode(), nil, nil, nil))
	}

	prompt := func() string {
		return "> "
	}

	processEvent := func(event model.Event) string {
		message := event.ToString(*currentChar)

		switch event.Type() {
		case model.RoomUpdateEventType:
			roomEvent := event.(model.RoomUpdateEvent)
			if roomEvent.Room.Id == currentRoom.Id {
				currentRoom = roomEvent.Room
			}
		}

		return message
	}

	eventChannel := model.Register(currentChar)
	defer model.Unregister(eventChannel)

	userInputChannel := make(chan string)
	promptChannel := make(chan string)

	inputModeChannel := make(chan userInputMode)
	panicChannel := make(chan interface{})

	/**
	 * Allows us to retrieve user input in a way that doesn't block the
	 * event loop by using channels and a separate Go routine to grab
	 * either the next user input or the next event.
	 */
	getUserInput := func(inputMode userInputMode, prompt string) string {
		inputModeChannel <- inputMode
		promptChannel <- prompt

		for {
			select {
			case input := <-userInputChannel:
				return input
			case event := <-eventChannel:
				message := processEvent(event)
				if message != "" {
					printLine("\n" + message)
					printString(prompt)
				}
			case quitMessage := <-panicChannel:
				panic(quitMessage)
			}
		}
		panic("Unexpected code path")
	}

	// Same behavior as menu.Exec(), except that it uses getUserInput
	// which doesn't block the event loop while waiting for input
	execMenu := func(menu utils.Menu) (string, bson.ObjectId) {
		choice := ""
		var data bson.ObjectId

		for {
			menu.Print(conn, currentUser.GetColorMode())
			choice = getUserInput(CleanUserInput, menu.GetPrompt())
			if menu.HasAction(choice) || choice == "" {
				data = menu.GetData(choice)
				break
			}
		}
		return choice, data
	}

	processAction := func(action string, args []string) {
		switch action {
		case "l":
			fallthrough
		case "look":
			if len(args) == 0 {
				printRoom()
			} else if len(args) == 1 {
				arg := database.StringToDirection(args[0])

				if arg == database.DirectionNone {
					printLine("Nothing to see")
				} else {
					loc := currentRoom.NextLocation(arg)
					roomToSee, found := model.M.GetRoomByLocation(loc, currentZone.Id)
					if found {
						printLine(roomToSee.ToString(database.ReadMode, currentUser.GetColorMode(),
							model.M.CharactersIn(roomToSee.Id, ""), model.M.NpcsIn(roomToSee.Id), nil))
					} else {
						printLine("Nothing to see")
					}
				}
			}

		case "ls":
			printLine("Where do you think you are?!")

		case "inventory":
			fallthrough
		case "inv":
			fallthrough
		case "i":
			itemIds := currentChar.GetItemIds()

			if len(itemIds) == 0 {
				printLine("You aren't carrying anything")
			} else {
				var itemNames []string
				for _, item := range model.M.GetItems(itemIds) {
					itemNames = append(itemNames, item.PrettyName())
				}
				printLine("You are carrying: %s", strings.Join(itemNames, ", "))
			}

			printLine("Cash: %v", currentChar.GetCash())

		case "take":
			fallthrough
		case "t":
			fallthrough
		case "get":
			fallthrough
		case "g":
			takeUsage := func() {
				printError("Usage: take <item name>")
			}

			if len(args) != 1 {
				takeUsage()
				return
			}

			itemsInRoom := model.M.GetItems(currentRoom.GetItemIds())
			for _, item := range itemsInRoom {
				if item.PrettyName() == args[0] {
					currentChar.AddItem(item)
					currentRoom.RemoveItem(item)
					return
				}
			}

			printError("Item %s not found", args[0])

		case "drop":
			dropUsage := func() {
				printError("Usage: drop <item name>")
			}

			if len(args) != 1 {
				dropUsage()
				return
			}

			characterItems := model.M.GetItems(currentChar.GetItemIds())

			for _, item := range characterItems {
				if item.PrettyName() == args[0] {
					currentChar.RemoveItem(item)
					currentRoom.AddItem(item)
					return
				}
			}

			printError("You are not carrying a %s", args[0])

		case "":
			fallthrough
		case "logout":
			return

		case "quit":
			fallthrough
		case "exit":
			printLine("Take luck!")
			panic("User quit")

		default:
			direction := database.StringToDirection(action)

			if direction != database.DirectionNone {
				if currentRoom.HasExit(direction) {
					newRoom, err := model.MoveCharacter(currentChar, direction)
					if err == nil {
						currentRoom = newRoom
						printRoom()
					} else {
						printError(err.Error())
					}

				} else {
					printError("You can't go that way")
				}
			} else {
				printError("You can't do that")
			}
		}
	}

	processCommand := func(command string, args []string) {
		switch command {
		case "?":
			fallthrough
		case "help":
		case "edit":
			printRoomEditor()

			for {
				input := getUserInput(CleanUserInput, "Select a section to edit> ")

				switch input {
				case "":
					printRoom()
					return

				case "1":
					input = getUserInput(RawUserInput, "Enter new title: ")

					if input != "" {
						currentRoom.SetTitle(input)
					}
					printRoomEditor()

				case "2":
					input = getUserInput(RawUserInput, "Enter new description: ")

					if input != "" {
						currentRoom.SetDescription(input)
					}
					printRoomEditor()

				case "3":
					for {
						menu := toggleExitMenu(currentUser.GetColorMode(), currentRoom)

						choice, _ := execMenu(menu)

						if choice == "" {
							break
						}

						direction := database.StringToDirection(choice)
						if direction != database.DirectionNone {
							enable := !currentRoom.HasExit(direction)
							currentRoom.SetExitEnabled(direction, enable)
						}
					}

					printRoomEditor()

				default:
					printLine("Invalid selection")
				}
			}

			// Quick room/exit creation
		case "/n":
			currentRoom.SetExitEnabled(database.DirectionNorth, true)
			processAction("n", []string{})
		case "/e":
			currentRoom.SetExitEnabled(database.DirectionEast, true)
			processAction("e", []string{})
		case "/s":
			currentRoom.SetExitEnabled(database.DirectionSouth, true)
			processAction("s", []string{})
		case "/w":
			currentRoom.SetExitEnabled(database.DirectionWest, true)
			processAction("w", []string{})
		case "/u":
			currentRoom.SetExitEnabled(database.DirectionUp, true)
			processAction("u", []string{})
		case "/d":
			currentRoom.SetExitEnabled(database.DirectionDown, true)
			processAction("d", []string{})

		case "loc":
			fallthrough
		case "location":
			printLine("%v", currentRoom.GetLocation())

		case "map":
			mapUsage := func() {
				printError("Usage: /map [<radius>|all|load <name>]")
			}

			startX := 0
			startY := 0
			startZ := 0
			endX := 0
			endY := 0
			endZ := 0

			if len(args) == 0 {
				args = append(args, "10")
			}

			if len(args) == 1 {
				radius, err := strconv.Atoi(args[0])

				if err == nil && radius > 0 {
					startX = currentRoom.GetLocation().X - radius
					startY = currentRoom.GetLocation().Y - radius
					startZ = currentRoom.GetLocation().Z
					endX = startX + (radius * 2)
					endY = startY + (radius * 2)
					endZ = currentRoom.GetLocation().Z
				} else if args[0] == "all" {
					topLeft, bottomRight := model.ZoneCorners(currentZone.Id)

					startX = topLeft.X
					startY = topLeft.Y
					startZ = topLeft.Z
					endX = bottomRight.X
					endY = bottomRight.Y
					endZ = bottomRight.Z
				} else {
					mapUsage()
					return
				}
			} else {
				mapUsage()
				return
			}

			width := endX - startX + 1
			height := endY - startY + 1
			depth := endZ - startZ + 1

			builder := newMapBuilder(width, height, depth)
			builder.setUserRoom(currentRoom)

			for z := startZ; z <= endZ; z += 1 {
				for y := startY; y <= endY; y += 1 {
					for x := startX; x <= endX; x += 1 {
						loc := database.Coordinate{x, y, z}
						currentRoom, found := model.M.GetRoomByLocation(loc, currentZone.Id)

						if found {
							// Translate to 0-based coordinates and double the coordinate
							// space to leave room for the exit lines
							builder.addRoom(currentRoom, (x-startX)*2, (y-startY)*2, z-startZ)
						}
					}
				}
			}

			printLine(utils.TrimEmptyRows(builder.toString(currentUser.GetColorMode())))

		case "zone":
			if len(args) == 0 {
				if currentZone.Id == "" {
					printLine("Currently in the null zone")
				} else {
					printLine("Current zone: " + utils.Colorize(currentUser.GetColorMode(), utils.ColorBlue, currentZone.Name))
				}
			} else if len(args) == 1 {
				if args[0] == "list" {
					printLineColor(utils.ColorBlue, "Zones")
					printLineColor(utils.ColorBlue, "-----")
					for _, zone := range model.M.GetZones() {
						printLine(zone.Name)
					}
				} else {
					printError("Usage: /zone [list|rename <name>|new <name>]")
				}
			} else if len(args) == 2 {
				if args[0] == "rename" {
					_, found := model.M.GetZoneByName(args[0])

					if found {
						printError("A zone with that name already exists")
						return
					}

					if currentZone.Id == "" {
						currentZone = model.M.CreateZone(args[1])
						model.MoveRoomsToZone("", currentZone.Id)
					} else {
						currentZone.SetName(args[1])
					}
				} else if args[0] == "new" {
					_, found := model.M.GetZoneByName(args[0])

					if found {
						printError("A zone with that name already exists")
						return
					}

					newZone := model.M.CreateZone(args[1])
					newRoom := model.M.CreateRoom(newZone.Id)

					model.MoveCharacterToRoom(currentChar, newRoom)

					currentZone = newZone
					currentRoom = newRoom

					printRoom()
				}
			}

		case "broadcast":
			fallthrough
		case "b":
			if len(args) == 0 {
				printError("Nothing to say")
			} else {
				model.BroadcastMessage(*currentChar, strings.Join(args, " "))
			}

		case "say":
			fallthrough
		case "s":
			if len(args) == 0 {
				printError("Nothing to say")
			} else {
				model.Say(*currentChar, strings.Join(args, " "))
			}

		case "me":
			model.Emote(*currentChar, strings.Join(args, " "))

		case "whisper":
			fallthrough
		case "tell":
			fallthrough
		case "w":
			if len(args) < 2 {
				printError("Usage: /whisper <player> <message>")
				return
			}

			name := string(args[0])
			targetChar, found := model.M.GetCharacterByName(name)

			if !found {
				printError("Player '%s' not found", name)
				return
			}

			if !targetChar.IsOnline() {
				printError("Player '%s' is not online", targetChar.PrettyName())
				return
			}

			message := strings.Join(args[1:], " ")
			model.Tell(currentChar, targetChar, message)

		case "teleport":
			fallthrough
		case "tel":
			telUsage := func() {
				printError("Usage: /teleport [<zone>|<X> <Y> <Z>]")
			}

			x := 0
			y := 0
			z := 0

			newZone := currentZone

			if len(args) == 1 {
				var found bool
				newZone, found = model.M.GetZoneByName(args[0])

				if !found {
					printError("Zone not found")
					return
				}

				if newZone.Id == currentRoom.GetZoneId() {
					printLine("You're already in that zone")
					return
				}

				zoneRooms := model.M.GetRoomsInZone(newZone.Id)

				if len(zoneRooms) > 0 {
					r := zoneRooms[0]
					x = r.GetLocation().X
					y = r.GetLocation().Y
					z = r.GetLocation().Z
				}
			} else if len(args) == 3 {
				var err error
				x, err = strconv.Atoi(args[0])

				if err != nil {
					telUsage()
					return
				}

				y, err = strconv.Atoi(args[1])

				if err != nil {
					telUsage()
					return
				}

				z, err = strconv.Atoi(args[2])

				if err != nil {
					telUsage()
					return
				}
			} else {
				telUsage()
				return
			}

			newRoom, err := model.MoveCharacterToLocation(currentChar, newZone.Id, database.Coordinate{X: x, Y: y, Z: z})

			if err == nil {
				currentRoom = newRoom
				currentZone = newZone
				printRoom()
			} else {
				printError(err.Error())
			}

		case "who":
			chars := model.M.GetOnlineCharacters()

			printLine("")
			printLine("Online Players")
			printLine("--------------")

			for _, char := range chars {
				printLine(char.PrettyName())
			}
			printLine("")

		case "colors":
			printLineColor(utils.ColorRed, "Red")
			printLineColor(utils.ColorDarkRed, "Dark Red")
			printLineColor(utils.ColorGreen, "Green")
			printLineColor(utils.ColorDarkGreen, "Dark Green")
			printLineColor(utils.ColorBlue, "Blue")
			printLineColor(utils.ColorDarkBlue, "Dark Blue")
			printLineColor(utils.ColorYellow, "Yellow")
			printLineColor(utils.ColorDarkYellow, "Dark Yellow")
			printLineColor(utils.ColorMagenta, "Magenta")
			printLineColor(utils.ColorDarkMagenta, "Dark Magenta")
			printLineColor(utils.ColorCyan, "Cyan")
			printLineColor(utils.ColorDarkCyan, "Dark Cyan")
			printLineColor(utils.ColorBlack, "Black")
			printLineColor(utils.ColorWhite, "White")
			printLineColor(utils.ColorGray, "Gray")

		case "colormode":
			fallthrough
		case "cm":
			if len(args) == 0 {
				message := "Current color mode is: "
				switch currentUser.GetColorMode() {
				case utils.ColorModeNone:
					message = message + "None"
				case utils.ColorModeLight:
					message = message + "Light"
				case utils.ColorModeDark:
					message = message + "Dark"
				}
				printLine(message)
			} else if len(args) == 1 {
				switch strings.ToLower(args[0]) {
				case "none":
					currentUser.SetColorMode(utils.ColorModeNone)
					printLine("Color mode set to: None")
				case "light":
					currentUser.SetColorMode(utils.ColorModeLight)
					printLine("Color mode set to: Light")
				case "dark":
					currentUser.SetColorMode(utils.ColorModeDark)
					printLine("Color mode set to: Dark")
				default:
					printLine("Valid color modes are: None, Light, Dark")
				}
			} else {
				printLine("Valid color modes are: None, Light, Dark")
			}

		case "delete":
			fallthrough
		case "del":
			if len(args) == 1 {
				direction := database.StringToDirection(args[0])

				if direction == database.DirectionNone {
					printError("Not a valid direction")
				} else {
					loc := currentRoom.NextLocation(direction)
					roomToDelete, found := model.M.GetRoomByLocation(loc, currentZone.Id)
					if found {
						model.DeleteRoom(roomToDelete)
					} else {
						printError("No room in that direction")
					}
				}
			} else {
				printError("Usage: /delete <direction>")
			}

		case "npc":
			menu := npcMenu(currentRoom)
			choice, npcId := execMenu(menu)

			getName := func() string {
				name := ""
				for {
					name = getUserInput(CleanUserInput, "Desired NPC name: ")
					_, found := model.M.GetCharacterByName(name)

					if name == "" {
						return ""
					} else if found {
						printError("That name is unavailable")
					} else if err := utils.ValidateName(name); err != nil {
						printError(err.Error())
					} else {
						break
					}
				}
				return name
			}

			if choice == "" {
				goto done
			}

			if choice == "n" {
				name := getName()
				if name == "" {
					goto done
				}
				database.NewNpc(name, currentRoom.Id)
			} else if npcId != "" {
				specificMenu := specificNpcMenu(npcId)
				choice, _ := execMenu(specificMenu)

				switch choice {
				case "d":
					model.M.DeleteCharacter(npcId)
				case "r":
					name := getName()
					if name == "" {
						break
					}
					npc := model.M.GetCharacter(npcId)
					npc.SetName(name)
				}
			}

		done:
			printRoom()

		case "create":
			createUsage := func() {
				printError("Usage: /create <item name>")
			}

			if len(args) != 1 {
				createUsage()
				return
			}

			item := model.M.CreateItem(args[0])
			currentRoom.AddItem(item)

		case "destroy":
			destroyUsage := func() {
				printError("Usage: /destroy <item name>")
			}

			if len(args) != 1 {
				destroyUsage()
				return
			}

			itemsInRoom := model.M.GetItems(currentRoom.GetItemIds())

			for _, item := range itemsInRoom {
				if item.PrettyName() == args[0] {
					currentRoom.RemoveItem(item)
					model.M.DeleteItem(item.Id)

					printLine("Item destroyed")
					return
				}
			}

			printError("Item not found")

		case "cash":
			cashUsage := func() {
				printError("Usage: /cash give <amount>")
			}

			if len(args) != 2 {
				cashUsage()
				return
			}

			if args[0] == "give" {
				amount, err := strconv.Atoi(args[1])

				if err != nil {
					cashUsage()
					return
				}

				currentChar.AddCash(amount)
				printLine("Received: %v monies", amount)
			} else {
				cashUsage()
				return
			}

		case "roomid":
			printLine("Room ID: %v", currentRoom.GetId())

		default:
			printError("Unrecognized command: %s", command)
		}
	}

	printLineColor(utils.ColorWhite, "Welcome, "+currentChar.PrettyName())
	printRoom()

	// Main routine in charge of actually reading input from the connection object,
	// also has built in throttling to limit how fast we are allowed to process
	// commands from the user. 
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChannel <- r
			}
		}()

		lastTime := time.Now()

		delay := time.Duration(200) * time.Millisecond

		for {
			mode := <-inputModeChannel
			prompt := utils.Colorize(currentUser.GetColorMode(), utils.ColorWhite, <-promptChannel)
			input := ""

			switch mode {
			case CleanUserInput:
				input = utils.GetUserInput(conn, prompt)
			case RawUserInput:
				input = utils.GetRawUserInput(conn, prompt)
			default:
				panic("Unhandled case in switch statement (userInputMode)")
			}

			diff := time.Since(lastTime)

			if diff < delay {
				time.Sleep(delay - diff)
			}

			userInputChannel <- input
			lastTime = time.Now()
		}
	}()

	// Main loop
	for {
		input := getUserInput(RawUserInput, prompt())
		if input == "" {
			return
		}
		if strings.HasPrefix(input, "/") {
			processCommand(utils.Argify(input[1:]))
		} else {
			processAction(utils.Argify(input))
		}
	}
}

// vim: nocindent
