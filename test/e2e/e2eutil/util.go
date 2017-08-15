package e2eutil

// PodLabelForOperator returns a label of the form name=<name>
func PodLabelForOperator(name string) map[string]string {
	return map[string]string{"name": name}
}
