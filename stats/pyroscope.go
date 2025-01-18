package stats

import (
	"errors"
	"os"
	"runtime"

	"github.com/grafana/pyroscope-go"
)

func InitPyroscope(serverAddress string) (*pyroscope.Profiler, error) {
	if serverAddress == "" {
		return nil, errors.New("pyroscope server address is required")
	}

	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	return pyroscope.Start(pyroscope.Config{
		ApplicationName: "magnetico.tgragnato.it",
		ServerAddress:   serverAddress,
		Logger:          nil,
		Tags:            map[string]string{"hostname": os.Getenv("HOSTNAME")},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
}
