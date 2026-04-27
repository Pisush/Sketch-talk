package audio

import (
	"context"
	"fmt"

	"github.com/gen2brain/malgo"
)

const (
	SampleRate = 16000
	Channels   = 1
)

// StartCapture opens the audio device and streams int16 PCM samples to sampleCh.
// It returns when ctx is cancelled.
func StartCapture(ctx context.Context, deviceIndex int, sampleCh chan<- []int16) error {
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("malgo init context: %w", err)
	}
	defer mctx.Uninit()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = uint32(Channels)
	deviceConfig.SampleRate = uint32(SampleRate)
	deviceConfig.Alsa.NoMMap = 1

	if deviceIndex >= 0 {
		devices, err := mctx.Devices(malgo.Capture)
		if err == nil && deviceIndex < len(devices) {
			deviceConfig.Capture.DeviceID = devices[deviceIndex].ID.Pointer()
		}
	}

	onRecv := func(_, pSamples []byte, frameCount uint32) {
		samples := make([]int16, frameCount*uint32(Channels))
		for i := range samples {
			lo := pSamples[i*2]
			hi := pSamples[i*2+1]
			samples[i] = int16(uint16(lo) | uint16(hi)<<8)
		}
		select {
		case sampleCh <- samples:
		default: // drop if consumer is slow
		}
	}

	callbacks := malgo.DeviceCallbacks{Data: onRecv}
	device, err := malgo.InitDevice(mctx.Context, deviceConfig, callbacks)
	if err != nil {
		return fmt.Errorf("malgo init device: %w", err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		return fmt.Errorf("malgo start device: %w", err)
	}
	defer device.Stop()

	<-ctx.Done()
	return nil
}
