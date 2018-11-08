package scamp

// agent receives updates to itself in the form of functions, via it's inbox
type actor struct {
	inbox chan<- actorFunc
}

// closure consumed by agent via inbox
type actorFunc func()

//actor must implement an inbox loop example:
// func (a *DiscoveryAnnouncer) inboxLoop(actionChan <-chan actorFunc) {
// 	Info.Println("starting discovery agent inboxLoop")
// inbox:
// 	for {
// 		select {
// 		case <-a.stopSig:
// 			Warning.Println("received stopSig")
// 			break inbox
// 		case action := <-actionChan:
// 			action()
// 		default:
// 		}
// 	}
// 	Warning.Println("discovery agent inboxLoop stopped")
// }
