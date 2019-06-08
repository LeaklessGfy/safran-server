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
	buffers sync.Map
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
		o.buffers.Store(measure.Inc, make([]byte, 0))
	}
	o.length = len(measures)
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
		former, ok := o.buffers.Load(sample.Inc)
		if !ok {
			return errors.New("Can't locate " + strconv.Itoa(sample.Inc) + " entry")
		}
		formerBytes, ok := former.([]byte)
		if !ok {
			return errors.New("Can't convert " + strconv.Itoa(sample.Inc) + " entry to bytes")
		}
		newBytes := append(formerBytes, append(b, byte(','))...)
		o.buffers.Store(sample.Inc, newBytes)
	}

	o.buffers.Range(func(k, v interface{}) bool {
		bytes, ok := v.([]byte)
		if !ok {
			return false
		}
		key, ok := k.(int)
		if !ok {
			return false
		}
		if len(bytes) > 1000 {
			o.group.Add(1)
			go func() {
				err := o.flushBuffer(key)
				if err != nil {
					log.Println("[CONCURRENT]", err)
				}
				o.group.Done()
			}()
		}
		return true
	})

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

func (o *JSONOutput) flushBuffer(index int) error {
	f, err := os.OpenFile("./dumps/"+strconv.Itoa(index)+".json", os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	defer f.Close()
	it, ok := o.buffers.Load(index)
	if !ok {
		return errors.New("Can't locate " + strconv.Itoa(index) + " inside map")
	}
	bytes, ok := it.([]byte)
	if !ok {
		return errors.New("Can't convert value of map to bytes")
	}
	_, err = f.Write(bytes)
	if err != nil {
		return err
	}
	o.buffers.Store(index, make([]byte, 0))
	return nil
}
