package profilter

import (
	"errors"

	"github.com/getlantern/http-proxy-lantern/redis"
	redislib "gopkg.in/redis.v3"
)

type proConfig struct {
	serverId    string
	redisConfig *redis.ProConfig
	userTokens  redis.UserTokens
	proFilter   *lanternProFilter
}

func NewRedisProConfig(rc *redislib.Client, serverId string, proFilter *lanternProFilter) *proConfig {
	return &proConfig{
		serverId:    serverId,
		redisConfig: redis.NewProConfig(rc, serverId),
		userTokens:  make(redis.UserTokens),
		proFilter:   proFilter,
	}
}

func (c *proConfig) processUserSetMessage(msg []string) error {
	// Should receive USER-SET,<USER>,<TOKEN>
	if len(msg) != 3 {
		return errors.New("Malformed SET message")
	}
	user := msg[1]
	token := msg[2]
	c.userTokens[user] = token
	return nil
}

func (c *proConfig) processUserRemoveMessage(msg []string) error {
	// Should receive USER-REMOVE,<USER>
	if len(msg) != 2 {
		return errors.New("Malformed REMOVE message")
	}
	user := msg[1]
	if _, ok := c.userTokens[user]; !ok {
		return errors.New("User in REMOVE message was not assigned to server")
	}

	delete(c.userTokens, user)
	return nil
}

func (c *proConfig) getAllTokens() []string {
	tokens := make([]string, len(c.userTokens))
	i := 0
	for _, v := range c.userTokens {
		tokens[i] = v
		i++
	}
	return tokens
}

func (c *proConfig) IsPro() (bool, error) {
	return c.redisConfig.IsPro()
}

func (c *proConfig) Run(initAsPro bool) error {
	initialize := func() (err error) {
		if c.userTokens, err = c.redisConfig.GetUsersAndTokens(); err != nil {
			return
		}

		// Initialize only if there are users assigned to this server
		if len(c.userTokens) > 0 {
			c.proFilter.Enable()
		} else {
			return
		}

		tks := c.getAllTokens()
		c.proFilter.SetTokens(tks...)
		log.Debugf("Initializing with the following Pro tokens: %v", tks)
		return
	}

	if initAsPro {
		if err := initialize(); err != nil {
			return err
		}
	}

	go func() {
		for {
			msg, err := c.redisConfig.GetNextMessage()
			if err != nil {
				log.Debugf("Error reading message from Redis: %v", err)
				continue
			}
			switch msg[0] {
			case "USER-SET":
				c.redisConfig.EmptyMessageQueue()
				// If this is the first user of the proxy, initialization will be required
				if len(c.userTokens) == 0 {
					initialize()
				}
				// Add or update an user
				if err := c.processUserSetMessage(msg); err != nil {
					log.Errorf("Error setting user/token: %v", err)
				} else {
					// We need to update all tokens to avoid leaking old ones,
					// in case of token update
					c.proFilter.SetTokens(c.getAllTokens()...)
					log.Tracef("User added/updated. Complete set of users: %v", c.userTokens)
				}
			case "USER-REMOVE":
				if err := c.processUserRemoveMessage(msg); err != nil {
					log.Errorf("Error retrieving removed users/token: %v", err)
				} else {
					c.proFilter.SetTokens(c.getAllTokens()...)
					log.Tracef("Removed user. Current set: %v", c.userTokens)
				}
			case "TURN-PRO":
				initialize()
				log.Debug("Proxy now is Pro-only. Tokens updated.")
			case "TURN-FREE":
				c.proFilter.Disable()
				c.proFilter.ClearTokens()
				log.Debug("Proxy now is Free-only")
			default:
				log.Error("Unknown message type")
			}
		}
	}()
	return nil
}
