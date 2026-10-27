package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	seaplanev1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1"
	"github.com/openshift-online/rosa-seaplane-api/seaplane-api/pkg/clients/seaplanedb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// placeholderMCAccountID is NOT a real AWS account. ManagementCluster.Spec.AccountID is
// required by the CRD but nothing reads it: placement and the status stream key on the
// name. Startup registration is temporary until the MC autoscaler registers clusters.
const placeholderMCAccountID = "000000000000"

// managementClusters parses MANAGEMENT_CLUSTERS, comma-separated entries in the regional
// Terraform format ("name:account"), into the clusters to register here. Only the name
// is used; the account suffix is ignored.
func managementClusters(value, region string) ([]*seaplanev1alpha1.ManagementCluster, error) {
	var clusters []*seaplanev1alpha1.ManagementCluster
	for entry := range strings.SplitSeq(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, _, _ := strings.Cut(entry, ":")
		if name == "" {
			return nil, fmt.Errorf("invalid MANAGEMENT_CLUSTERS entry %q: want name or name:account", entry)
		}
		clusters = append(clusters, &seaplanev1alpha1.ManagementCluster{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       seaplanev1alpha1.ManagementClusterSpec{Region: region, AccountID: placeholderMCAccountID},
		})
	}
	return clusters, nil
}

// registerManagementClusters writes each cluster as the registration endpoint did.
// Existing clusters are kept, so restarts are safe.
func registerManagementClusters(ctx context.Context, db *seaplanedb.Client, clusters []*seaplanev1alpha1.ManagementCluster, logger *slog.Logger) error {
	for _, mc := range clusters {
		if err := db.CreateManagementCluster(ctx, mc); err != nil && !seaplanedb.IsAlreadyExists(err) {
			return fmt.Errorf("failed to register management cluster %s: %w", mc.Name, err)
		}
		logger.Info("management cluster registered", "name", mc.Name)
	}
	return nil
}
