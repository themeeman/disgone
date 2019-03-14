package disgone

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"strings"
	"time"
)

func getCommand(commands CommandMap, handlers HandlerMap, name string) (*Command, Handler, string) {
	name = strings.ToLower(name)
	command, okc := commands[name]
	handler, okh := handlers[name]
	if !okc || !okh {
		for n, cmd := range commands {
			for _, alias := range cmd.Aliases {
				if name == alias {
					return getCommand(commands, handlers, n)
				}
			}
		}
	}
	return command, handler, name
}

func getExecuteFunc(options *BotOptions, handlers HandlerMap, userGroup UserGroup) func(session *discordgo.Session, message *discordgo.MessageCreate) {
	var execute func(*discordgo.Session, *discordgo.MessageCreate)
	execute = func(session *discordgo.Session, message *discordgo.MessageCreate) {
		defer func() {
			if r := recover(); r != nil {
				options.OnPanic(session, message, r)
			}
		}()
		t := time.Now()
		if !hasPrefix(message.Content, options.Prefix) {
			return
		}
		args := strings.Fields(trimPrefix(message.Content, options.Prefix))
		if len(args) == 0 {
			return
		}
		info, cmd, name := getCommand(options.Commands, handlers, args[0])
		if cmd == nil {
			return
		}
		g, _ := session.Guild(mustGetGuildID(session, message))
		if g == nil {
			return
		}
		m, _ := session.GuildMember(g.ID, message.Author.ID)
		if m == nil {
			return
		}
		if userGroup != nil {
			if level := userGroup(session, g, m); level < info.Group {
				if options.Errors != nil {
					options.Errors <- struct {
						Err error
						*discordgo.MessageCreate
					}{
						Err: InsufficientPermissionsError{
							Required: options.GroupNames[info.Group],
							Had:      options.GroupNames[level],
						},
						MessageCreate: message,
					}
				}
				return
			}
		}
		args = args[1:]
		if len(args) == 0 && info != nil && info.ForcedArgs() > 0 {
			if options.Errors != nil {
				options.Errors <- struct {
					Err error
					*discordgo.MessageCreate
				}{
					Err:           ZeroArgumentsError{Command: name},
					MessageCreate: message,
				}
			}
			return
		}
		var newArgs = map[string]string{}
		if options.Types != nil {
			var err error
			newArgs, err = parseArgs(options.Types, info, args)
			if err != nil {
				if options.Errors != nil {
					if e, ok := err.(UsageError); ok {
						e.Usage = info.GetUsage(options.Prefix, name)
						err = e
					}
					options.Errors <- struct {
						Err error
						*discordgo.MessageCreate
					}{
						Err:           err,
						MessageCreate: message,
					}
				}
				return
			}
		}
		fmt.Println(newArgs)
		err := cmd(session, message, newArgs)
		if err != nil {
			if options.Errors != nil {
				options.Errors <- struct {
					Err error
					*discordgo.MessageCreate
				}{
					Err:           err,
					MessageCreate: message,
				}
			}
			return
		}
		fmt.Println(time.Since(t))
		return
	}
	return execute
}

func NewSession(bot interface{}, options *BotOptions, token string) (*discordgo.Session, error) {
	t := reflect.TypeOf(bot)
	v := reflect.ValueOf(bot)
	var handlers = make(HandlerMap)
	fmt.Println(t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		funcValue := v.Method(i)
		funcType := v.Method(i).Type()
		handlerType := reflect.TypeOf(Handler(nil))
		if funcType.ConvertibleTo(handlerType) {
			fmt.Println(strings.ToLower(t.Method(i).Name))
			handlers[strings.ToLower(t.Method(i).Name)] = funcValue.Convert(handlerType).Interface().(Handler)
		}
	}
	var userGroup func(session *discordgo.Session, guild *discordgo.Guild, member *discordgo.Member) Group
	if _, ok := t.MethodByName("UserGroup"); ok {
		if f, ok := v.MethodByName("UserGroup").Interface().(func(session *discordgo.Session, guild *discordgo.Guild, member *discordgo.Member) Group); ok {
			userGroup = f
		}
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	execute := getExecuteFunc(options, handlers, userGroup)
	dg.AddHandler(execute)
	return dg, nil
}


func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.ToLower(s)[0:len(prefix)] == strings.ToLower(prefix)
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func mustGetGuildID(session *discordgo.Session, message *discordgo.MessageCreate) string {
	c, err := session.Channel(message.ChannelID)
	if err != nil {
		return ""
	}
	return c.GuildID
}