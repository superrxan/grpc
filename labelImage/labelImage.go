// Copyright 2021 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package labelImage

import (
	"bufio"
	"bytes"
	"fmt"
	tflite "github.com/mattn/go-tflite"
	"github.com/nfnt/resize"
	"grpc/message"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sort"
	"sync"
)

type labelImage struct {
	modelPath   string
	labelPath   string
	once        sync.Once
	interpreter *tflite.Interpreter
	labels      []string
}

var LabelImage = labelImage{
	modelPath: "labelImage/etc/mobilenet_quant_v1_224.tflite",
	labelPath: "labelImage/etc/labels.txt",
}

func (f *labelImage) Init() {
	f.labels, _ = loadLabels(f.labelPath)

	model := tflite.NewModelFromFile(f.modelPath)
	if model == nil {
		panic(fmt.Errorf("fail to load model: %s", f.modelPath))
		return
	}
	defer model.Delete()

	options := tflite.NewInterpreterOptions()
	options.SetNumThread(4)
	options.SetErrorReporter(func(msg string, userData interface{}) {
		fmt.Println(msg)
	}, nil)
	defer options.Delete()

	interpreter := tflite.NewInterpreter(model, options)
	if interpreter == nil {
		panic(fmt.Errorf("cannot create interpreter"))
		return
	}
	status := interpreter.AllocateTensors()
	if status != tflite.OK {
		interpreter.Delete()
		panic(fmt.Errorf("allocate failed"))
		return
	}

	f.interpreter = interpreter
	// TODO If created, the interpreter will be kept through the whole life of kuiper. Refactor this later.
	//defer interpreter.Delete()
}

func (f *labelImage) Exec(args []byte) (*message.LabelReply, error) {
	img, _, err := image.Decode(bytes.NewReader(args))
	if err != nil {
		return nil, err
	}

	input := f.interpreter.GetInputTensor(0)
	wantedHeight := input.Dim(1)
	wantedWidth := input.Dim(2)
	wantedChannels := input.Dim(3)
	wantedType := input.Type()

	resized := resize.Resize(uint(wantedWidth), uint(wantedHeight), img, resize.NearestNeighbor)
	bounds := resized.Bounds()
	dx, dy := bounds.Dx(), bounds.Dy()

	if wantedType == tflite.UInt8 {
		bb := make([]byte, dx*dy*wantedChannels)
		for y := 0; y < dy; y++ {
			for x := 0; x < dx; x++ {
				col := resized.At(x, y)
				r, g, b, _ := col.RGBA()
				bb[(y*dx+x)*3+0] = byte(float64(r) / 255.0)
				bb[(y*dx+x)*3+1] = byte(float64(g) / 255.0)
				bb[(y*dx+x)*3+2] = byte(float64(b) / 255.0)
			}
		}
		input.CopyFromBuffer(bb)
	} else {
		return nil, fmt.Errorf("is not wanted type")
	}

	status := f.interpreter.Invoke()
	if status != tflite.OK {
		return nil, fmt.Errorf("invoke failed")
	}

	output := f.interpreter.GetOutputTensor(0)
	outputSize := output.Dim(output.NumDims() - 1)
	b := make([]byte, outputSize)
	type result struct {
		score float32
		index int
	}
	status = output.CopyToBuffer(&b[0])
	if status != tflite.OK {
		return nil, fmt.Errorf("output failed")
	}
	var results []result
	for i := 0; i < outputSize; i++ {
		score := float32(b[i]) / 255.0
		if score < 0.2 {
			continue
		}
		results = append(results, result{score: score, index: i})

	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var reply = &message.LabelReply{
	}

	for _, v := range results {
		re := &message.LabelResult{
			Confidence: v.score,
			Label:      f.labels[v.index],
		}
		reply.Results = append(reply.Results, re)
	}
	return reply, nil
}

func loadLabels(filename string) ([]string, error) {
	var labels []string
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		labels = append(labels, scanner.Text())
	}
	return labels, nil
}
