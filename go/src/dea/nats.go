package dea

import (
	cfg "dea/config"
	"github.com/cloudfoundry/yagnats"
	"github.com/nu7hatch/gouuid"
	"strconv"
)

type Nats struct {
	NatsClient yagnats.NATSClient
	config     cfg.NatsConfig
	sids       []int
}

type NatsHandler interface {
	HandleHealthManagerStart(msg *yagnats.Message)
	HandleRouterStart(msg *yagnats.Message)
	HandleDeaStatus(msg *yagnats.Message)
	HandleDeaDirectedStart(msg *yagnats.Message)
	HandleDeaStop(msg *yagnats.Message)
	HandleDeaUpdate(msg *yagnats.Message)
	HandleDeaFindDroplet(msg *yagnats.Message)
	UUID() string
}

func NewNats(config cfg.NatsConfig) *Nats {
	return &Nats{
		NatsClient: yagnats.NewClient(),
		config:     config,
		sids:       make([]int, 0, 7),
	}
}

func (n *Nats) Start(handler NatsHandler) error {
	addr := n.config.Host + ":" + strconv.FormatUint(uint64(n.config.Port), 10)
	connection := &yagnats.ConnectionInfo{
		Addr:     addr,
		Username: n.config.User,
		Password: n.config.Pass,
	}

	if err := n.NatsClient.Connect(connection); err != nil {
		return err
	}

	if err := n.subscribe("healthmanager.start", handler.HandleHealthManagerStart); err != nil {
		return err
	}

	if err := n.subscribe("router.start", handler.HandleRouterStart); err != nil {
		return err
	}

	if err := n.subscribe("dea.status", handler.HandleDeaStatus); err != nil {
		return err
	}

	uuid := handler.UUID()
	if err := n.subscribe("dea."+uuid+".start", handler.HandleDeaDirectedStart); err != nil {
		return err
	}

	if err := n.subscribe("dea.stop", handler.HandleDeaStop); err != nil {
		return err
	}

	if err := n.subscribe("dea.update", handler.HandleDeaUpdate); err != nil {
		return err
	}

	if err := n.subscribe("dea.find.droplet", handler.HandleDeaFindDroplet); err != nil {
		return err
	}

	return nil
}

func (n *Nats) subscribe(subject string, callback yagnats.Callback) error {
	sid, err := n.NatsClient.Subscribe(subject, callback)
	if err != nil {
		return err
	}

	n.sids = append(n.sids, sid)

	return nil
}

func (n *Nats) Request(subject string, message []byte, callback yagnats.Callback) (int, error) {
	inbox, err := n.createInbox()
	if err != nil {
		return 0, err
	}

	sid, err := n.NatsClient.Subscribe(inbox, callback)
	if err != nil {
		return sid, err
	}

	err = n.NatsClient.PublishWithReplyTo(subject, inbox, message)
	return sid, err
}

func (n *Nats) Stop() {
	for _, sid := range n.sids {
		n.NatsClient.Unsubscribe(sid)
	}
	n.sids = make([]int, 0, 1)
}

func (n *Nats) createInbox() (string, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	return "_INBOX." + uuid.String(), nil
}
