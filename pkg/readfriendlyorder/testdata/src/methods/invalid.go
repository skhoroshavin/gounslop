package methods

type Worker struct{}

func (w *Worker) doWork() int { return 1 } // want `Place method "doWork" below method "Process" that depends on it.`

func (w *Worker) Process() int {
	return w.doWork()
}
