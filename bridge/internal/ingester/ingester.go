package ingester

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Sample struct {
	Time         time.Time `json:"time"`
	Collector    string    `json:"collector"`
	Plant        string    `json:"plant"`
	Temperature  *float32  `json:"temp"`
	Light        *float32  `json:"light"`
	Moisture     *float32  `json:"moist"`
	Conductivity *float32  `json:"cond"`
	Humidity     *float32  `json:"humid"`
	Battery      *int      `json:"battery"`
}

const timeout = 10 * time.Second

type Ingester struct {
	url    string
	client *http.Client
}

func NewIngester(url string) *Ingester {
	return &Ingester{
		url: url,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (i *Ingester) SendAll(ctx context.Context, c <-chan Sample) error {
	logrus.Debug("starting to sending samples")

	for {
		select {
		case <-ctx.Done():
			logrus.Debug("ingester context cancelled")
			return nil
		case m := <-c:
			err := i.send(m)
			if err != nil {
				logrus.Error(err)
			}
		}
	}
}

func (i *Ingester) send(m Sample) error {
	logrus.Debugf("sending sample %v", m)

	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}

	logrus.WithField("data", string(jsonData)).Debug("sending sample")

	request, err := http.NewRequest("POST", i.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := i.client.Do(request)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
