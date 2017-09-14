# Status Conditions and Events

We need to provide observability to Vault operator users.
This will give user insights and help when debugging Vault operator

The method should also be user-friendly and k8s native.
Currently most k8s users would use `kubectl describe` and inspect status conditions and events.
We would obey the same rules and expose internal states similarly.

## Conditions

```Go
type VaultServiceConditionType string

// These are valid conditions of a vault service.
const (
	// Available means the vault service is available, ie. an active node exists.
	VaultServiceAvailable VaultServiceConditionType = "Available"
	// Progressing means the vault service is progressing.
	// A vault service is marked progressing when one of the following tasks is performed:
	// - Upgrade is happening and nothing is blocked. If upgrade is blocked on waiting users
	//   to unseal new nodes, progressing is set to "False" with a reason.
	VaultServiceProgressing VaultServiceConditionType = "Progressing"
	// ReplicaFailure is added in a vault service when one of its pods fails to be created
	// or deleted.
	VaultServiceReplicaFailure VaultServiceConditionType = "ReplicaFailure"
)

// VaultServiceCondition describes the state of a vault service at a certain point.
type VaultServiceCondition struct {
	// Type of vault service condition.
	Type VaultServiceConditionType
	// Status of the condition: True, False, or Unknown.
	Status api.ConditionStatus
	// The last time this condition was updated.
	LastUpdateTime metav1.Time
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time
	// The reason for the condition's last transition.
	Reason string
	// A human readable message indicating details about the transition.
	Message string
}

```

## Events

Events:
- Vault is initialized
- Vault node X become active
- Vault node X become standby
- Vault upgrade initiated
- Vault upgrade finished
