package output

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/leaklessgfy/safran-server/entity"
)

type JSONOutput struct {
	date    time.Time
	length  int
	buffers [][]byte
	group   *sync.WaitGroup
}

func (o *JSONOutput) SaveExperiment(experiment *entity.Experiment) error {
	o.date = experiment.StartDate
	o.group = &sync.WaitGroup{}
	err := os.RemoveAll("./dumps")
	if err != nil {
		return err
	}
	return os.Mkdir("./dumps", 0777)
}

func (o *JSONOutput) SaveMeasures(measures []*entity.Measure) error {
	o.length = len(measures)
	o.buffers = make([][]byte, o.length)
	for _, measure := range measures {
		f, err := os.Create("./dumps/" + strconv.Itoa(measure.Inc) + ".json")
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write([]byte(`{"measure":"` + measure.Name + `","type":"` + measure.Typex + `","unit":"` + measure.Unitx + `","values":[`))
		if err != nil {
			return err
		}
		o.buffers[measure.Inc] = make([]byte, 0)
	}
	return nil
}

func (o *JSONOutput) SaveSamples(samples []*entity.Sample) error {
	for _, sample := range samples {
		b, err := json.Marshal(struct {
			Time  string `json:"time"`
			Value string `json:"value"`
		}{
			Time:  sample.Time,
			Value: sample.Value,
		})
		if err != nil {
			return err
		}
		if sample.Inc >= o.length {
			return errors.New("sample index > measures length, index=" + strconv.Itoa(sample.Inc) + ", length=" + strconv.Itoa(o.length))
		}
		o.buffers[sample.Inc] = append(o.buffers[sample.Inc], append(b, byte(','))...)
	}

	var keys []int
	for key, buffer := range o.buffers {
		if len(buffer) > 1000 {
			o.group.Add(1)
			go func() {
				err := flushBuffer(key, buffer)
				if err != nil {
					log.Println("[CONCURRENT]", err)
				}
				o.group.Done()
			}()
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		o.buffers[key] = make([]byte, 0)
	}

	return nil
}

func (o JSONOutput) SaveAlarms([]*entity.Alarm) error {
	return nil
}

func (o JSONOutput) Cancel() error {
	return os.RemoveAll("./dumps")
}

func (o JSONOutput) End() error {
	o.group.Wait()
	for i := 0; i < o.length; i++ {
		f, err := os.OpenFile("./dumps/"+strconv.Itoa(i)+".json", os.O_RDWR, 0777)
		if err != nil {
			return err
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			return err
		}
		offset := stat.Size() - 1
		b := make([]byte, 1)
		l, err := f.ReadAt(b, offset)
		if err != nil || l < 1 {
			return err
		}
		if b[0] != byte(',') {
			offset = offset + 1
		}
		_, err = f.WriteAt([]byte("]}"), offset)
		if err != nil {
			return err
		}
	}
	return nil
}

func flushBuffer(index int, buffer []byte) error {
	f, err := os.OpenFile("./dumps/"+strconv.Itoa(index)+".json", os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buffer)
	if err != nil {
		return err
	}
	return nil
}
