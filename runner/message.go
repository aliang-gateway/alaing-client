package runner

import "os"

var TunSignal = make(chan os.Signal, 1)
var RunStatusChan = make(chan map[string]string, 1)
