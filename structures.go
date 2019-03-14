package disgone

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type BotOptions struct {
	Commands   CommandMap
	Types      map[string]string
	Prefix     string
	GroupNames map[Group]string
	Errors     chan struct {
		Err error
		*discordgo.MessageCreate
	}
	OnPanic func(*discordgo.Session, *discordgo.MessageCreate, interface{})
}

type Argument struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
	Infinite bool   `json:"infinite"`
}

type Command struct {
	Description string      `json:"description"`
	Arguments   []*Argument `json:"arguments"`
	Cooldown    int         `json:"cooldown"`
	Group       `json:"permission"`
	Aliases     []string `json:"aliases"`
	Examples    []string `json:"examples"`
	Hidden      bool     `json:"hidden"`
}

type Handler func(session *discordgo.Session,
	message *discordgo.MessageCreate,
	args map[string]string) error

type UserGroup func(session *discordgo.Session,
	guild *discordgo.Guild,
	member *discordgo.Member) Group

type HandlerMap map[string]Handler

type CommandMap map[string]*Command

type Group uint

func (c Command) ForcedArgs() (i int) {
	for _, v := range c.Arguments {
		if !v.Optional {
			i += 1
		}
	}
	return
}

func (c Command) GetUsage(prefix, name string) string {
	var buffer bytes.Buffer
	buffer.WriteString(prefix + name)
	for _, v := range c.Arguments {
		if v.Optional {
			buffer.WriteString(fmt.Sprintf(" <%s>", v.Name))
		} else {
			buffer.WriteString(fmt.Sprintf(" [%s]", v.Name))
		}
	}
	return buffer.String()
}
