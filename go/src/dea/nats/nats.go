package nats

import (
	"dea"
	"github.com/cloudfoundry/yagnats"
	"github.com/nu7hatch/gouuid"
	"net/url"
)

type Nats struct {
	NatsClient  yagnats.NATSClient
	connectInfo yagnats.ConnectionCluster
	sids        []int
}

func NewNats(servers []string) (*Nats, error) {
	members := make([]yagnats.ConnectionProvider, 0, len(servers))
	for _, server := range servers {
		natsURL, err := url.Parse(server)
		if err != nil {
			return nil, err
		}

		username := ""
		password := ""
		if natsURL.User != nil {
			username = natsURL.User.Username()
			password, _ = natsURL.User.Password()
		}

		members = append(members, &yagnats.ConnectionInfo{
			Addr:     natsURL.Host,
			Username: username,
			Password: password,
		})
	}

	connectionCluster := yagnats.ConnectionCluster{
		Members: members,
	}

	return &Nats{
		NatsClient:  yagnats.NewClient(),
		connectInfo: connectionCluster,
		sids:        make([]int, 0, 7),
	}, nil
}

func (n *Nats) Client() yagnats.NATSClient {
	return n.NatsClient
}

func (n *Nats) Start(handler dea.NatsHandler) error {
	if err := n.NatsClient.Connect(&n.connectInfo); err != nil {
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

func (n *Nats) Unsubscribe() {
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
