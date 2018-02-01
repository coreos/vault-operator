package e2e

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"
	"github.com/coreos-inc/vault-operator/test/e2e/e2eutil"
	"github.com/coreos-inc/vault-operator/test/e2e/framework"

	eopapi "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/coreos/etcd-operator/pkg/util/etcdutil"
	eopk8sutil "github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"github.com/coreos/etcd-operator/pkg/util/retryutil"
	eope2eutil "github.com/coreos/etcd-operator/test/e2e/e2eutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getEndpoints returns endpoints of an etcd cluster given the cluster name and the residing namespace.
func getEndpoints(kubeClient kubernetes.Interface, secureClient bool, namespace, clusterName string) ([]string, error) {
	podList, err := kubeClient.Core().Pods(namespace).List(eopk8sutil.ClusterListOpt(clusterName))
	if err != nil {
		return nil, err
	}

	var pods []*v1.Pod
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase == v1.PodRunning {
			pods = append(pods, pod)
		}
	}

	if len(pods) == 0 {
		return nil, errors.New("no running etcd pods found")
	}

	endpoints := make([]string, len(pods))
	for i, pod := range pods {
		m := &etcdutil.Member{
			Name:         pod.Name,
			Namespace:    pod.Namespace,
			SecureClient: secureClient,
		}
		endpoints[i] = m.ClientURL()
	}
	return endpoints, nil
}

func createBackup(t *testing.T, vaultCR *api.VaultService, etcdClusterName, s3Path string) {
	f := framework.Global
	endpoints, err := getEndpoints(f.KubeClient, true, f.Namespace, etcdClusterName)
	if err != nil {
		t.Fatalf("failed to get endpoints: %v", err)
	}
	backupCR := eope2eutil.NewS3Backup(endpoints, etcdClusterName, s3Path, os.Getenv("TEST_AWS_SECRET"), k8sutil.EtcdClientTLSSecretName(vaultCR.Name))
	eb, err := f.EtcdCRClient.EtcdV1beta2().EtcdBackups(f.Namespace).Create(backupCR)
	if err != nil {
		t.Fatalf("failed to create etcd backup cr: %v", err)
	}
	defer func() {
		if err := f.EtcdCRClient.EtcdV1beta2().EtcdBackups(f.Namespace).Delete(eb.Name, nil); err != nil {
			t.Fatalf("failed to delete etcd backup cr: %v", err)
		}
	}()

	// local testing shows that it takes around 1 - 2 seconds from creating backup cr to verifying the backup from s3.
	// 4 seconds timeout via retry is enough; duration longer than that may indicate internal issues and
	// is worthy of investigation.
	err = retryutil.Retry(time.Second, 4, func() (bool, error) {
		reb, err := f.EtcdCRClient.EtcdV1beta2().EtcdBackups(f.Namespace).Get(eb.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to retrieve backup CR: %v", err)
		}
		if reb.Status.Succeeded {
			if reb.Status.EtcdVersion == eopapi.DefaultEtcdVersion && reb.Status.EtcdRevision > 1 {
				return true, nil
			}
			return false, fmt.Errorf("expect EtcdVersion==%v and EtcdRevision > 1, but got EtcdVersion==%v and EtcdRevision==%v", eopapi.DefaultEtcdVersion, reb.Status.EtcdVersion, reb.Status.EtcdRevision)
		}
		if len(reb.Status.Reason) != 0 {
			return false, fmt.Errorf("backup failed with reason: %v ", reb.Status.Reason)
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed to verify backup: %v", err)
	}
	t.Logf("backup for cluster (%s) has been saved", etcdClusterName)
}

func killEtcdCluster(t *testing.T, etcdClusterName string) {
	f := framework.Global
	lops := eopk8sutil.ClusterListOpt(etcdClusterName)
	err := f.KubeClient.CoreV1().Pods(f.Namespace).DeleteCollection(metav1.NewDeleteOptions(0), lops)
	if err != nil {
		t.Fatalf("failed to delete etcd cluster pods: %v", err)
	}
	if _, err := e2eutil.WaitPodsDeletedCompletely(f.KubeClient, f.Namespace, 6, lops); err != nil {
		t.Fatalf("failed to see the etcd cluster pods to be completely removed: %v", err)
	}
}

// restoreEtcdCluster restores an etcd cluster with name "etcdClusterName" from a backup saved on "s3Path".
func restoreEtcdCluster(t *testing.T, s3Path, etcdClusterName string) {
	f := framework.Global
	restoreSource := eopapi.RestoreSource{S3: eope2eutil.NewS3RestoreSource(s3Path, os.Getenv("TEST_AWS_SECRET"))}
	er := eope2eutil.NewEtcdRestore(etcdClusterName, 3, restoreSource, eopapi.BackupStorageTypeS3)
	er, err := f.EtcdCRClient.EtcdV1beta2().EtcdRestores(f.Namespace).Create(er)
	if err != nil {
		t.Fatalf("failed to create etcd restore cr: %v", err)
	}
	defer func() {
		if err := f.EtcdCRClient.EtcdV1beta2().EtcdRestores(f.Namespace).Delete(er.Name, nil); err != nil {
			t.Fatalf("failed to delete etcd restore cr: %v", err)
		}
	}()

	err = retryutil.Retry(10*time.Second, 1, func() (bool, error) {
		er, err := f.EtcdCRClient.EtcdV1beta2().EtcdRestores(f.Namespace).Get(er.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to retrieve restore CR: %v", err)
		}
		if er.Status.Succeeded {
			return true, nil
		} else if len(er.Status.Reason) != 0 {
			return false, fmt.Errorf("restore failed with reason: %v ", er.Status.Reason)
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed to verify restore succeeded: %v", err)
	}

	// Verify that the restored etcd cluster scales to 3 ready members
	restoredCluster := &eopapi.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdClusterName,
			Namespace: f.Namespace,
		},
		Spec: eopapi.ClusterSpec{
			Size: 3,
		},
	}
	if _, err := eope2eutil.WaitUntilSizeReached(t, f.EtcdCRClient, 3, 6, restoredCluster); err != nil {
		t.Fatalf("failed to see restored etcd cluster(%v) reach 3 members: %v", restoredCluster.Name, err)
	}
}

// verifyRestoredVault ensures that the vault cluster that's restored from an earlier backup contains the correct data:
func verifyRestoredVault(t *testing.T, vaultCR *api.VaultService, secretData map[string]interface{}, keyPath, rootToken string) {
	f := framework.Global

	vaultCR, tlsConfig := e2eutil.WaitForCluster(t, f.KubeClient, f.VaultsCRClient, vaultCR)
	vaultCR, err := e2eutil.WaitActiveVaultsUp(t, f.VaultsCRClient, 6, vaultCR)
	if err != nil {
		t.Fatalf("failed to wait for any node to become active: %v", err)
	}

	podName := vaultCR.Status.VaultStatus.Active
	vClient := e2eutil.SetupVaultClient(t, f.KubeClient, f.Namespace, tlsConfig, podName)
	vClient.SetToken(rootToken)
	e2eutil.VerifySecretData(t, vClient, secretData, keyPath, podName)
}

func TestBackupRestoreOnVault(t *testing.T) {
	f := framework.Global
	s3Path := path.Join(os.Getenv("TEST_S3_BUCKET"), "jenkins", strconv.Itoa(int(rand.Uint64())), time.Now().Format(time.RFC3339), "etcd.backup")

	vaultCR, tlsConfig, rootToken := e2eutil.SetupUnsealedVaultCluster(t, f.KubeClient, f.VaultsCRClient, f.Namespace)
	defer func(vaultCR *api.VaultService) {
		if err := e2eutil.DeleteCluster(t, f.VaultsCRClient, vaultCR); err != nil {
			t.Fatalf("failed to delete vault cluster: %v", err)
		}
	}(vaultCR)
	vClient, keyPath, secretData, podName := e2eutil.WriteSecretData(t, vaultCR, f.KubeClient, tlsConfig, rootToken, f.Namespace)
	e2eutil.VerifySecretData(t, vClient, secretData, keyPath, podName)

	etcdClusterName := k8sutil.EtcdNameForVault(vaultCR.Name)
	createBackup(t, vaultCR, etcdClusterName, s3Path)

	killEtcdCluster(t, etcdClusterName)
	restoreEtcdCluster(t, s3Path, etcdClusterName)
	verifyRestoredVault(t, vaultCR, secretData, keyPath, rootToken)
}
