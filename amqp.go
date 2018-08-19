package logspout_amqp

import (
	"github.com/gliderlabs/logspout/router"
	"github.com/streadway/amqp"
	"encoding/json"
	"time"
	"os"
	"log"
	"reflect"
	"text/template"
	"bytes"
	
)

func init() {
	router.AdapterFactories.Register(NewAmqpAdapter, "amqp")
}

type AmqpAdapter struct {
	route         *router.Route
	address       string
	exchange      string
	exchange_type string
	key           string
	user          string
	password      string
	tmpl *template.Template
}

func NewAmqpAdapter(route *router.Route) (router.LogAdapter, error) {
	address := route.Address

	// get our config value from the environment
	key := getEnv("AMQP_ROUTING_KEY", "logspout")
	exchange := getEnv("AMQP_EXCHANGE", "logs")
	exchange_type := getEnv("AMQP_EXCHANGE_TYPE", "direct")
	user := getEnv("AMQP_USER", "guest")
	password := getEnv("AMQP_PASSWORD", "guest")
	tmplStr := getEnv("RAW_FORMAT", "{{.Data}}\n")

	return &AmqpAdapter{
		route:         route,
		address:       address,
		exchange:      exchange,
		exchange_type: exchange_type,
		key:           key,
		user:          user,
		password:      password,
		tmpl:	       tmpl,
	}, nil

}

func (a *AmqpAdapter) Stream(logstream chan *router.Message) {
	// Open AMQP connection to the URI
	connection, err := amqp.Dial("amqp://" + a.user + ":" + a.password + "@" + a.address)
	failOnError(err, "amqp.connection.open")

	// close the connection when function finishes
	defer connection.Close()

	// Open Channel on the connection
	channel, err := connection.Channel()
	failOnError(err, "amqp.channel.open")

	// Typically AMQP implementations would channel.ExchangeDeclare(foo bar etc)
	// but we are assuming the exchange is created on RabbitMQ already
	for message := range logstream {
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		
		if err != nil {
			log.Println("raw:", err)
			return
		}
		
		err = channel.Publish(a.exchange, a.key, false, false, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Priority:     0,
			Timestamp:    time.Now(),
			Body:         buf,
		})
		failOnError(err, "amqp.message.publish")
	}
}

func getEnv(envkey string, default_value string) (value string) {
	value = os.Getenv(envkey)
	if value == "" {
		value = default_value
	}
	return
}


func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
