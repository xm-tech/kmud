package database

import (
	"fmt"
	"kmud/utils"
	"labix.org/v2/mgo/bson"
	"strings"
)

type Coordinate struct {
	X int
	Y int
	Z int
}

// All database types should meet this interface
type Identifiable interface {
	GetId() bson.ObjectId
	GetType() objectType
}

type Nameable interface {
	PrettyName() string
}

type objectType int

const (
	characterType objectType = iota
)

const (
	dbObjectName string = "name"
)

type DbObject struct {
	Id      bson.ObjectId `bson:"_id"`
	objType objectType
	Name    string `bson:",omitempty"`
	Fields  map[string]interface{}
}

type User struct {
	DbObject  `bson:",inline"`
	ColorMode utils.ColorMode
	online    bool
}

func NewUser(name string) User {
	var user User
	user.Id = bson.NewObjectId()
	user.Name = name
	user.ColorMode = utils.ColorModeNone
	return user
}

type ObjectField string

type Zone struct {
	DbObject `bson:",inline"`
}

func NewZone(name string) Zone {
	var zone Zone
	zone.Id = bson.NewObjectId()
	zone.Name = name
	return zone
}

type Room struct {
	DbObject      `bson:",inline"`
	ZoneId        bson.ObjectId `bson:",omitempty"`
	Title         string
	Description   string
	Items         []bson.ObjectId
	Location      Coordinate
	ExitNorth     bool
	ExitNorthEast bool
	ExitEast      bool
	ExitSouthEast bool
	ExitSouth     bool
	ExitSouthWest bool
	ExitWest      bool
	ExitNorthWest bool
	ExitUp        bool
	ExitDown      bool
	Default       bool
}

func NewRoom(zoneId bson.ObjectId) Room {
	var room Room
	room.Id = bson.NewObjectId()
	room.Title = "The Void"
	room.Description = "You are floating in the blackness of space. Complete darkness surrounds " +
		"you in all directions. There is no escape, there is no hope, just the emptiness. " +
		"You are likely to be eaten by a grue."

	room.ExitNorth = false
	room.ExitNorthEast = false
	room.ExitEast = false
	room.ExitSouthEast = false
	room.ExitSouth = false
	room.ExitSouthWest = false
	room.ExitWest = false
	room.ExitNorthWest = false
	room.ExitUp = false
	room.ExitDown = false

	room.Location = Coordinate{0, 0, 0}

	room.Default = false

	room.ZoneId = zoneId

	return room
}

type Item struct {
	DbObject `bson:",inline"`
}

func NewItem(name string) Item {
	var item Item
	item.Id = bson.NewObjectId()
	item.Name = name

	return item
}

type ExitDirection int

const (
	DirectionNone      ExitDirection = iota
	DirectionNorth     ExitDirection = iota
	DirectionNorthEast ExitDirection = iota
	DirectionEast      ExitDirection = iota
	DirectionSouthEast ExitDirection = iota
	DirectionSouth     ExitDirection = iota
	DirectionSouthWest ExitDirection = iota
	DirectionWest      ExitDirection = iota
	DirectionNorthWest ExitDirection = iota
	DirectionUp        ExitDirection = iota
	DirectionDown      ExitDirection = iota
)

type PrintMode int

const (
	ReadMode PrintMode = iota
	EditMode PrintMode = iota
)

func directionToExitString(colorMode utils.ColorMode, direction ExitDirection) string {

	letterColor := utils.ColorBlue
	bracketColor := utils.ColorDarkBlue
	textColor := utils.ColorWhite

	colorize := func(letters string, text string) string {
		return fmt.Sprintf("%s%s%s%s",
			utils.Colorize(colorMode, bracketColor, "["),
			utils.Colorize(colorMode, letterColor, letters),
			utils.Colorize(colorMode, bracketColor, "]"),
			utils.Colorize(colorMode, textColor, text))
	}

	switch direction {
	case DirectionNorth:
		return colorize("N", "orth")
	case DirectionNorthEast:
		return colorize("NE", "North East")
	case DirectionEast:
		return colorize("E", "ast")
	case DirectionSouthEast:
		return colorize("SE", "South East")
	case DirectionSouth:
		return colorize("S", "outh")
	case DirectionSouthWest:
		return colorize("SW", "South West")
	case DirectionWest:
		return colorize("W", "est")
	case DirectionNorthWest:
		return colorize("NW", "North West")
	case DirectionUp:
		return colorize("U", "p")
	case DirectionDown:
		return colorize("D", "own")
	case DirectionNone:
		return utils.Colorize(colorMode, utils.ColorWhite, "None")
	}

	panic("Unexpected code path")
}

func (self *DbObject) initDbObject(objType objectType) {
	self.objType = objType
	self.Fields = map[string]interface{}{}
}

func (self DbObject) GetId() bson.ObjectId {
	return self.Id
}

func (self DbObject) GetType() objectType {
	return self.objType
}

func (self DbObject) PrettyName() string {
	return utils.FormatName(self.Name)
}

func (self *DbObject) setField(key string, value interface{}) {
	self.Fields[key] = value
	updateObject(*self, "fields."+key, value)
}

func (self *DbObject) SetName(name string) {
	self.Name = name
	updateObject(*self, dbObjectName, name)
}

func (self *Room) ToString(mode PrintMode, colorMode utils.ColorMode, chars []*Character, npcs []*Character, items []Item) string {
	var str string

	if mode == ReadMode {
		str = fmt.Sprintf("\n %v %v %v (%v %v %v)\n\n %v\n\n",
			utils.Colorize(colorMode, utils.ColorWhite, ">>>"),
			utils.Colorize(colorMode, utils.ColorBlue, self.Title),
			utils.Colorize(colorMode, utils.ColorWhite, "<<<"),
			self.Location.X,
			self.Location.Y,
			self.Location.Z,
			utils.Colorize(colorMode, utils.ColorWhite, self.Description))

		extraNewLine := ""

		if len(chars) > 0 {
			str = str + " " + utils.Colorize(colorMode, utils.ColorBlue, "Also here: ")

			var names []string
			for _, char := range chars {
				names = append(names, utils.Colorize(colorMode, utils.ColorWhite, char.PrettyName()))
			}
			str = str + strings.Join(names, utils.Colorize(colorMode, utils.ColorBlue, ", ")) + "\n"

			extraNewLine = "\n"
		}

		if len(npcs) > 0 {
			str = str + " " + utils.Colorize(colorMode, utils.ColorBlue, "NPCs: ")

			var names []string
			for _, npc := range npcs {
				names = append(names, utils.Colorize(colorMode, utils.ColorWhite, npc.PrettyName()))
			}
			str = str + strings.Join(names, utils.Colorize(colorMode, utils.ColorBlue, ", ")) + "\n"

			extraNewLine = "\n"
		}

		if len(items) > 0 {
			str = str + " " + utils.Colorize(colorMode, utils.ColorBlue, "Items: ")

			var names []string
			for _, item := range items {
				names = append(names, utils.Colorize(colorMode, utils.ColorWhite, item.PrettyName()))
			}
			str = str + strings.Join(names, utils.Colorize(colorMode, utils.ColorBlue, ", ")) + "\n"

			extraNewLine = "\n"
		}

		str = str + extraNewLine + " " + utils.Colorize(colorMode, utils.ColorBlue, "Exits: ")

	} else {
		str = fmt.Sprintf(" [1] %v \n\n [2] %v \n\n [3] Exits: ", self.Title, self.Description)
	}

	var exitList []string

	appendIfExists := func(direction ExitDirection) {
		if self.HasExit(direction) {
			exitList = append(exitList, directionToExitString(colorMode, direction))
		}
	}

	appendIfExists(DirectionNorth)
	appendIfExists(DirectionNorthEast)
	appendIfExists(DirectionEast)
	appendIfExists(DirectionSouthEast)
	appendIfExists(DirectionSouth)
	appendIfExists(DirectionSouthWest)
	appendIfExists(DirectionWest)
	appendIfExists(DirectionNorthWest)
	appendIfExists(DirectionUp)
	appendIfExists(DirectionDown)

	if len(exitList) == 0 {
		str = str + "None"
	} else {
		str = str + strings.Join(exitList, " ")
	}

	str = str + "\n"

	return str
}

func (self *Room) HasExit(dir ExitDirection) bool {
	switch dir {
	case DirectionNorth:
		return self.ExitNorth
	case DirectionNorthEast:
		return self.ExitNorthEast
	case DirectionEast:
		return self.ExitEast
	case DirectionSouthEast:
		return self.ExitSouthEast
	case DirectionSouth:
		return self.ExitSouth
	case DirectionSouthWest:
		return self.ExitSouthWest
	case DirectionWest:
		return self.ExitWest
	case DirectionNorthWest:
		return self.ExitNorthWest
	case DirectionUp:
		return self.ExitUp
	case DirectionDown:
		return self.ExitDown
	}

	panic("Unexpected code path")
}

func (self *Room) SetExitEnabled(dir ExitDirection, enabled bool) {
	switch dir {
	case DirectionNorth:
		self.ExitNorth = enabled
	case DirectionNorthEast:
		self.ExitNorthEast = enabled
	case DirectionEast:
		self.ExitEast = enabled
	case DirectionSouthEast:
		self.ExitSouthEast = enabled
	case DirectionSouth:
		self.ExitSouth = enabled
	case DirectionSouthWest:
		self.ExitSouthWest = enabled
	case DirectionWest:
		self.ExitWest = enabled
	case DirectionNorthWest:
		self.ExitNorthWest = enabled
	case DirectionUp:
		self.ExitUp = enabled
	case DirectionDown:
		self.ExitDown = enabled
	}
}

func (self *Room) AddItem(item Item) {
	self.Items = append(self.Items, item.GetId())
}

func (self *Room) RemoveItem(item Item) {
	for i, myItemId := range self.Items {
		if myItemId == item.GetId() {
			// TODO: Potential memory leak. See http://code.google.com/p/go-wiki/wiki/SliceTricks
			self.Items = append(self.Items[:i], self.Items[i+1:]...)
			return
		}
	}
}

func (self *User) SetOnline(online bool) {
	self.online = online
}

func (self *User) Online() bool {
	return self.online
}

func (self *Coordinate) Next(direction ExitDirection) Coordinate {
	newCoord := *self
	switch direction {
	case DirectionNorth:
		newCoord.Y -= 1
	case DirectionNorthEast:
		newCoord.Y -= 1
		newCoord.X += 1
	case DirectionEast:
		newCoord.X += 1
	case DirectionSouthEast:
		newCoord.Y += 1
		newCoord.X += 1
	case DirectionSouth:
		newCoord.Y += 1
	case DirectionSouthWest:
		newCoord.Y += 1
		newCoord.X -= 1
	case DirectionWest:
		newCoord.X -= 1
	case DirectionNorthWest:
		newCoord.Y -= 1
		newCoord.X -= 1
	case DirectionUp:
		newCoord.Z -= 1
	case DirectionDown:
		newCoord.Z += 1
	}
	return newCoord
}

func StringToDirection(str string) ExitDirection {
	dirStr := strings.ToLower(str)
	switch dirStr {
	case "n":
		fallthrough
	case "north":
		return DirectionNorth
	case "ne":
		return DirectionNorthEast
	case "e":
		fallthrough
	case "east":
		return DirectionEast
	case "se":
		return DirectionSouthEast
	case "s":
		fallthrough
	case "south":
		return DirectionSouth
	case "sw":
		return DirectionSouthWest
	case "w":
		fallthrough
	case "west":
		return DirectionWest
	case "nw":
		return DirectionNorthWest
	case "u":
		fallthrough
	case "up":
		return DirectionUp
	case "d":
		fallthrough
	case "down":
		return DirectionDown
	}

	return DirectionNone
}

// vim: nocindent
