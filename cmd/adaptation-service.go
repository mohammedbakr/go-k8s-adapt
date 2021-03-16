package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/k8-proxy/k8-go-comm/pkg/minio"
	"github.com/k8-proxy/k8-go-comm/pkg/rabbitmq"

	"github.com/streadway/amqp"

	miniov7 "github.com/minio/minio-go/v7"
)

const (
	ok        = "ok"
	jsonerr   = "json_error"
	k8sclient = "k8s_client_error"
	k8sapi    = "k8s_api_error"
)

var (
	exchange   = "adaptation-exchange"
	routingKey = "adaptation-request"
	queueName  = "adaptation-request-queue"

	podNamespace                          = "not-needed"
	metricsPort                           = "1234"
	inputMount                            = os.Getenv("INPUT_MOUNT")
	outputMount                           = os.Getenv("OUTPUT_MOUNT")
	requestProcessingImage                = os.Getenv("REQUEST_PROCESSING_IMAGE")
	requestProcessingTimeout              = os.Getenv("REQUEST_PROCESSING_TIMEOUT")
	adaptationRequestQueueHostname        = os.Getenv("ADAPTATION_REQUEST_QUEUE_HOSTNAME")
	adaptationRequestQueuePort            = os.Getenv("ADAPTATION_REQUEST_QUEUE_PORT")
	archiveAdaptationRequestQueueHostname = os.Getenv("ARCHIVE_ADAPTATION_QUEUE_REQUEST_HOSTNAME")
	archiveAdaptationRequestQueuePort     = os.Getenv("ARCHIVE_ADAPTATION_REQUEST_QUEUE_PORT")
	transactionEventQueueHostname         = os.Getenv("TRANSACTION_EVENT_QUEUE_HOSTNAME")
	transactionEventQueuePort             = os.Getenv("TRANSACTION_EVENT_QUEUE_PORT")
	messagebrokeruser                     = os.Getenv("MESSAGE_BROKER_USER")
	messagebrokerpassword                 = os.Getenv("MESSAGE_BROKER_PASSWORD")
	cpuLimit                              = os.Getenv("CPU_LIMIT")
	cpuRequest                            = os.Getenv("CPU_REQUEST")
	memoryLimit                           = os.Getenv("MEMORY_LIMIT")
	memoryRequest                         = os.Getenv("MEMORY_REQUEST")

	minioEndpoint     = os.Getenv("MINIO_ENDPOINT")
	minioAccessKey    = os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey    = os.Getenv("MINIO_SECRET_KEY")
	sourceMinioBucket = os.Getenv("MINIO_SOURCE_BUCKET")

	publisher   *amqp.Channel
	minioClient *miniov7.Client
)

func main() {
	if podNamespace == "" || metricsPort == "" || inputMount == "" || outputMount == "" {
		log.Fatalf("init failed: POD_NAMESPACE, METRICS_PORT, INPUT_MOUNT or OUTPUT_MOUNT environment variables not set")
	}

	if adaptationRequestQueueHostname == "" || archiveAdaptationRequestQueueHostname == "" || transactionEventQueueHostname == "" {
		log.Fatalf("init failed: ADAPTATION_REQUEST_QUEUE_HOSTNAME, ARCHIVE_ADAPTATION_QUEUE_REQUEST_HOSTNAME or TRANSACTION_EVENT_QUEUE_HOSTNAME environment variables not set")
	}

	if adaptationRequestQueuePort == "" || archiveAdaptationRequestQueuePort == "" || transactionEventQueuePort == "" {
		log.Fatalf("init failed: ADAPTATION_REQUEST_QUEUE_PORT, ARCHIVE_ADAPTATION_REQUEST_QUEUE_PORT or TRANSACTION_EVENT_QUEUE_PORT environment variables not set")
	}

	if cpuLimit == "" || cpuRequest == "" || memoryLimit == "" || memoryRequest == "" {
		log.Fatalf("init failed: CPU_LIMIT, CPU_REQUEST, MEMORY_LIMIT or MEMORY_REQUEST environment variables not set")
	}

	if messagebrokeruser == "" {
		messagebrokeruser = "guest"
	}

	if messagebrokerpassword == "" {
		messagebrokerpassword = "guest"
	}

	// Get a connection
	connection, err := rabbitmq.NewInstance(adaptationRequestQueueHostname, adaptationRequestQueuePort, messagebrokeruser, messagebrokerpassword)
	if err != nil {
		log.Fatalf("%s", err)
	}

	// Initiate a publisher on processing exchange
	publisher, err = rabbitmq.NewQueuePublisher(connection, "")
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer publisher.Close()

	// Start a consumer
	msgs, ch, err := rabbitmq.NewQueueConsumer(connection, queueName, exchange, routingKey)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer ch.Close()

	minioClient, err = minio.NewMinioClient(minioEndpoint, minioAccessKey, minioSecretKey, false)
	if err != nil {
		log.Fatalf("%s", err)
	}

	forever := make(chan bool)

	// Consume
	go func() {
		for d := range msgs {
			err := processMessage(d)
			if err != nil {
				log.Printf("Failed to process message: %v", err)
			}
		}
	}()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func processMessage(d amqp.Delivery) error {

	if d.Headers["file-id"] == nil ||
		d.Headers["source-file-presigned-url"] == nil {
		return fmt.Errorf("Headers value is nil")
	}

	fileID := d.Headers["file-id"].(string)
	sourceUrl := d.Headers["source-file-presigned-url"].(string)
	generateReport := "false"

	if d.Headers["generate-report"] != nil {
		generateReport = d.Headers["generate-report"].(string)
	}

	log.Printf("Received a message for file: %s", fileID)

	// These are now local directories, not persistent volumes anymore.
	localInput := filepath.Join("/tmp", "input", fileID)
	localOutput := filepath.Join("/tmp", "output", fileID)

	// Download the file to the local input location
	err := minio.DownloadObject(sourceUrl, localInput)
	if err != nil {
		return err
	}

	os.Setenv("FileId", fileID)
	os.Setenv("InputPath", localInput)
	os.Setenv("OutputPath", localOutput)
	os.Setenv("GenerateReport", generateReport)
	os.Setenv("ReplyTo", d.ReplyTo)
	os.Setenv("ProcessingTimeoutDuration", requestProcessingTimeout)
	os.Setenv("AdaptationRequestQueueHostname", adaptationRequestQueueHostname)
	os.Setenv("AdaptationRequestQueuePort", adaptationRequestQueuePort)
	os.Setenv("ArchiveAdaptationRequestQueueHostname", archiveAdaptationRequestQueueHostname)
	os.Setenv("ArchiveAdaptationRequestQueuePort", archiveAdaptationRequestQueuePort)
	os.Setenv("TransactionEventQueueHostname", transactionEventQueueHostname)
	os.Setenv("TransactionEventQueuePort", transactionEventQueuePort)
	os.Setenv("MessageBrokerUser", messagebrokeruser)
	os.Setenv("MessageBrokerPassword", messagebrokerpassword)

	// Rebuild the file
	cmd := exec.Command("dotnet", "Service.dll")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("File processing error : %s\n", err.Error())
	}
	fmt.Printf("File processing output : %s\n", out)

	// Upload the rebuilt file to Minio
	outputPresignedURL, err := minio.UploadAndReturnURL(minioClient, sourceMinioBucket, localOutput, time.Second*60*60*24)
	if err != nil {
		return err
	}
	d.Headers["output-presigned-url"] = outputPresignedURL

	// Send response to the Queue
	err = rabbitmq.PublishMessage(publisher, "", d.ReplyTo, d.Headers, []byte(""))
	if err != nil {
		return err
	}

	return nil
}
