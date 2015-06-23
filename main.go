package main // import "websecurify/go-aws-attach-volume"

// ---
// ---
// ---

import (
	"os"
	"log"
	"net"
	"time"
	"syscall"
	"net/http"
	"io/ioutil"
	"os/signal"
	
	// ---
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// ---
// ---
// ---

const (
	HTTP_TIMEOUT = time.Duration(3.0 * 1000) * time.Millisecond
)

// ---

var config = struct {
	Region string
	Device string
	VolumeID string
	InstanceID string
} {
	Region: os.Getenv("REGION"),
	Device: os.Getenv("DEVICE"),
	VolumeID: os.Getenv("VOLUME_ID"),
	InstanceID: os.Getenv("INSTANCE_ID"),
}

// ---

var globals = struct {
	InstanceID string
	EC2Service *ec2.EC2
} {
	InstanceID: getInstanceID(),
	EC2Service: getEC2Service(),
}

// ---
// ---
// ---

func getInstanceID() (string) {
	if config.InstanceID != "" {
		return config.InstanceID
	}
	
	// ---
	
	dial := func(network string, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, HTTP_TIMEOUT)
	}
	
	// ---
	
	transport := http.Transport{
		Dial: dial,
	}
	
	// ---
	
	client := http.Client{
		Transport: &transport,
	}
	
	// ---
	
	getRes, getResErr := client.Get("http://169.254.169.254/latest/meta-data/instance-id")
	
	if getResErr != nil {
		log.Fatal(getResErr)
	}
	
	// ---
	
	defer getRes.Body.Close()
	
	// ---
	
	body, bodyErr := ioutil.ReadAll(getRes.Body)
	
	if bodyErr != nil {
		log.Fatal(bodyErr)
	}
	
	// ---
	
	return string(body)
}

func getEC2Service() (*ec2.EC2) {
	return ec2.New(&aws.Config{Region: config.Region})
}

// ---
// ---
// ---

func getVolumeAttachmentInstanceID() (string) {
	describeVolumesRes, describeVolumesErr := globals.EC2Service.DescribeVolumes(&ec2.DescribeVolumesInput{
		VolumeIDs: []*string{
			aws.String(config.VolumeID),
		},
	})
	
	if describeVolumesErr != nil {
		log.Fatal(describeVolumesErr)
	}
	
	// ---
	
	log.Println("volume attachments queried")
	log.Println(awsutil.StringValue(describeVolumesRes))
	
	// ---
	
	volumes := describeVolumesRes.Volumes
	
	// ---
	
	var instanceID string
	
	// ---
	
	if len(volumes) > 0 {
		if volumes[0].Attachments != nil && len(volumes[0].Attachments) > 0 {
			instanceID = *volumes[0].Attachments[0].InstanceID
		}
	}
	
	// ---
	
	return instanceID
}

// ---

func attachVolume() {
	attachRes, attachErr := globals.EC2Service.AttachVolume(&ec2.AttachVolumeInput{
		VolumeID: aws.String(config.VolumeID),
		Device: aws.String(config.Device),
		InstanceID: aws.String(globals.InstanceID),
		DryRun: aws.Boolean(false),
	})
	
	if attachErr != nil {
		log.Fatal(attachErr)
	}
	
	// ---
	
	log.Println("volume attached")
	log.Println(awsutil.StringValue(attachRes))
}

func detachVolume() {
	detachRes, detachErr := globals.EC2Service.DetachVolume(&ec2.DetachVolumeInput{
		VolumeID: aws.String(config.VolumeID),
		Device: aws.String(config.Device),
		InstanceID: aws.String(globals.InstanceID),
		DryRun: aws.Boolean(false),
	})
	
	if detachErr != nil {
		log.Fatal(detachErr)
	}
	
	// ---
	
	log.Println("volume detached")
	log.Println(awsutil.StringValue(detachRes))
}

// ---
// ---
// ---

func main() {
	volumeAttachmentInstanceID := getVolumeAttachmentInstanceID()
	
	if volumeAttachmentInstanceID == "" {
		attachVolume()
	} else
	if volumeAttachmentInstanceID != globals.InstanceID {
		log.Fatal("volume ", config.VolumeID, " already attached to ", volumeAttachmentInstanceID)
	}
	
	// ---
	
	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	
	// ---
	
	signal.Notify(sigs, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	
	// ---
	
	go func() {
		<-sigs
		
		// ---
		
		detachVolume()
		
		// ---
		
		done <- true
	}()
	
	// ---
	
	<-done
}

// ---
