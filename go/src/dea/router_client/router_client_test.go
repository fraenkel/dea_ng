package router_client_test

import (
	"dea"
	"dea/config"
	"dea/nats"
	. "dea/router_client"
	"dea/starting"
	"dea/testhelpers"
	"encoding/json"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RouterClient", func() {

	var fakeNats *fakeyagnats.FakeYagnats
	var client dea.RouterClient
	var instance *starting.Instance

	uuid := "dea-efd-123"
	local_ip := "127.0.0.7"

	host := "127.5.5.8"
	port := uint32(1234)
	uri := "guid234.cf-apps.io"

	application_id := "5678"
	instance_host_port := 7890
	private_instance_id := "instance-id-123"
	application_uri := "my-super-app.cf-app.com"

	BeforeEach(func() {
		nats, _ := nats.NewNats([]string{"nats://user:password@myexample.com:4222"})
		fakeNats = fakeyagnats.New()
		nats.NatsClient = fakeNats

		config := &config.Config{Index: 5}

		attrs := testhelpers.Valid_instance_attributes(false)
		attrs["application_id"] = application_id
		attrs["instance_host_port"] = instance_host_port
		attrs["application_uris"] = []interface{}{application_uri}
		attrs["private_instance_id"] = private_instance_id
		instance = starting.NewInstance(attrs, config, nil, local_ip)

		client = NewRouterClient(config, nats, uuid, local_ip)
	})

	Describe("register_directory_server", func() {
		It("sends a correct nats message", func() {
			client.Register_directory_server(host, port, uri)

			Expect(len(fakeNats.PublishedMessages["router.register"])).Should(Equal(1))

			message := fakeNats.PublishedMessages["router.register"][0]
			rrMsg := make(map[string]interface{})
			err := json.Unmarshal(message.Payload, &rrMsg)
			Expect(err).To(BeNil())
			Expect(rrMsg["host"]).To(Equal(host))
			Expect(rrMsg["port"]).To(BeNumerically("==", port))
			Expect(rrMsg["uris"]).To(Equal([]interface{}{uri}))
			Expect(rrMsg["tags"]).To(Equal(map[string]interface{}{"component": "directory-server-5"}))
		})
	})

	Describe("unregister_directory_server", func() {
		It("sends a correct nats message", func() {
			client.Unregister_directory_server(host, port, uri)

			Expect(len(fakeNats.PublishedMessages["router.unregister"])).Should(Equal(1))

			message := fakeNats.PublishedMessages["router.unregister"][0]

			rrMsg := make(map[string]interface{})
			err := json.Unmarshal(message.Payload, &rrMsg)
			Expect(err).To(BeNil())
			Expect(rrMsg["host"]).To(Equal(host))
			Expect(rrMsg["port"]).To(BeNumerically("==", port))
			Expect(rrMsg["uris"]).To(Equal([]interface{}{uri}))
			Expect(rrMsg["tags"]).To(Equal(map[string]interface{}{"component": "directory-server-5"}))
		})
	})

	Describe("register_instance", func() {
		It("sends a correct nats message", func() {
			client.RegisterInstance(instance, map[string]interface{}{"uris": []string{uri}})

			Expect(len(fakeNats.PublishedMessages["router.register"])).Should(Equal(1))

			message := fakeNats.PublishedMessages["router.register"][0]
			rrMsg := make(map[string]interface{})
			err := json.Unmarshal(message.Payload, &rrMsg)
			Expect(err).To(BeNil())
			Expect(rrMsg["dea"]).To(Equal(uuid))
			Expect(rrMsg["app"]).To(Equal(application_id))
			Expect(rrMsg["host"]).To(Equal(local_ip))
			Expect(rrMsg["port"]).To(BeNumerically("==", instance_host_port))
			Expect(rrMsg["uris"]).To(Equal([]interface{}{uri}))
			Expect(rrMsg["tags"]).To(Equal(map[string]interface{}{"component": "dea-5"}))
			Expect(rrMsg["private_instance_id"]).To(Equal(private_instance_id))
		})

		Context("when uri is not passed", func() {
			It("uses application uri", func() {
				client.RegisterInstance(instance, nil)

				Expect(len(fakeNats.PublishedMessages["router.register"])).Should(Equal(1))

				message := fakeNats.PublishedMessages["router.register"][0]
				rrMsg := make(map[string]interface{})
				err := json.Unmarshal(message.Payload, &rrMsg)
				Expect(err).To(BeNil())

				Expect(rrMsg["uris"]).To(Equal([]interface{}{application_uri}))

			})
		})
	})

	Describe("unregister_instance", func() {
		It("sends a correct nats message", func() {
			client.UnregisterInstance(instance, map[string]interface{}{"uris": []string{uri}})

			Expect(len(fakeNats.PublishedMessages["router.unregister"])).Should(Equal(1))

			message := fakeNats.PublishedMessages["router.unregister"][0]
			rrMsg := make(map[string]interface{})
			err := json.Unmarshal(message.Payload, &rrMsg)
			Expect(err).To(BeNil())
			Expect(rrMsg["dea"]).To(Equal(uuid))
			Expect(rrMsg["app"]).To(Equal(application_id))
			Expect(rrMsg["host"]).To(Equal(local_ip))
			Expect(rrMsg["port"]).To(BeNumerically("==", instance_host_port))
			Expect(rrMsg["uris"]).To(Equal([]interface{}{uri}))
			Expect(rrMsg["tags"]).To(Equal(map[string]interface{}{"component": "dea-5"}))
			Expect(rrMsg["private_instance_id"]).To(Equal(private_instance_id))
		})

		Context("when uri is not passed", func() {
			It("uses application uri", func() {
				client.UnregisterInstance(instance, nil)

				Expect(len(fakeNats.PublishedMessages["router.unregister"])).Should(Equal(1))

				message := fakeNats.PublishedMessages["router.unregister"][0]
				rrMsg := make(map[string]interface{})
				err := json.Unmarshal(message.Payload, &rrMsg)
				Expect(err).To(BeNil())

				Expect(rrMsg["uris"]).To(Equal([]interface{}{application_uri}))

			})
		})
	})

	Describe("greet", func() {
		It("sends greet message", func() {
			client.Greet(nil)
			Expect(len(fakeNats.PublishedMessages["router.greet"])).Should(Equal(1))
		})
	})
})
