package scheduler

type Scheduler interface {
	SelectCandidateWorker()
	Score()
	Pick()
}

