package recovery

// StartGroupRecovery Starts a single recovery.
// func (r *Group) StartGroupRecovery() {
// 	for _, recovery := range r.Recoveries {
// 		recovery.GetTreeStruct()
// 	}
// }

func (r *Recovery) RunRecovery() {

}

// LegacyRecovery recovers files using legacy blockserver remote
func (r *Recovery) LegacyRecovery() {}

// Recovery recovers files using current blockserver remote
func (r *Recovery) Recovery() {}
