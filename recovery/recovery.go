package recovery

// StartGroupRecovery Starts a single recovery.
func (r *RecoveryGroup) StartGroupRecovery() {
	for _, recovery := range r.Recoveries {
		recovery.GetTreeStruct()
	}
}

func (r *Recovery) LegacyRecovery() {}
func (r *Recovery) Recovery()       {}
