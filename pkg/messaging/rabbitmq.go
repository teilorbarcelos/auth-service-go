package messaging

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"backend-go/pkg/config"
)

type AMQPChannel interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

type AMQPConnection interface {
	Channel() (AMQPChannel, error)
	Close() error
}

type ConnectionWrapper struct {
	*amqp.Connection
}

func (w *ConnectionWrapper) Channel() (AMQPChannel, error) {
	return w.Connection.Channel()
}

var RabbitConn AMQPConnection
var RabbitChannel AMQPChannel

var amqpDial = func(url string) (AMQPConnection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &ConnectionWrapper{conn}, nil
}
var logFatalf = log.Fatalf

func ConnectRabbitMQ() {
	if config.AppConfig.Environment == "test" {
		log.Println("RabbitMQ ignorado em ambiente de teste.")
		return
	}

	var err error
	RabbitConn, err = amqpDial(config.AppConfig.RabbitMQUrl)
	if err != nil {
		logFatalf("Falha ao conectar no RabbitMQ: %v", err)
	}

	RabbitChannel, err = RabbitConn.Channel()
	if err != nil {
		logFatalf("Falha ao abrir canal no RabbitMQ: %v", err)
	}

	log.Println("Conexão com RabbitMQ estabelecida com sucesso.")
}

func Publish(queueName string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	if RabbitChannel == nil {
		return nil // Ou retorne um erro se for mandatório
	}

	q, err := RabbitChannel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return RabbitChannel.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		})
}
