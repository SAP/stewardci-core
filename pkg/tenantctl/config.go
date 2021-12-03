package tenantctl

import (
	"context"
	"fmt"
	"math"
	"strconv"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	errors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clientConfig interface {
	GetTenantNamespacePrefix() string
	GetTenantNamespaceSuffixLength() uint8
	GetTenantRoleName() k8s.RoleName
}

const (
	tenantNamespaceSuffixLengthDefault uint8 = 6
	tenantNamespaceSuffixLengthMax     uint8 = 32
)

type clientConfigImpl struct {
	tenantNamespacePrefix       string
	tenantNamespaceSuffixLength int64
	tenantRoleName              k8s.RoleName
}

// getClientConfig returns the configurartion of the Steward client.
func getClientConfig(ctx context.Context, factory k8s.ClientFactory, clientNamespace string) (clientConfig, error) {
	if clientNamespace == "" {
		return nil, errors.New("client namespace must not be empty")
	}

	newConfig := clientConfigImpl{
		tenantNamespaceSuffixLength: -1,
	}

	namespace, err := factory.CoreV1().Namespaces().Get(ctx, clientNamespace, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithMessagef(err, "could not get namespace '%s'", clientNamespace)
	}

	annotations := namespace.GetAnnotations()
	var value string
	var hasKey bool

	value, hasKey = annotations[steward.AnnotationTenantNamespacePrefix]
	if !hasKey {
		return nil, errors.Errorf("annotation '%s' is missing on client namespace '%s'", steward.AnnotationTenantNamespacePrefix, clientNamespace)
	}
	if value == "" {
		return nil, errors.Errorf("annotation '%s' on client namespace '%s' must not have an empty value", steward.AnnotationTenantNamespacePrefix, clientNamespace)
	}
	newConfig.tenantNamespacePrefix = value

	value, hasKey = annotations[steward.AnnotationTenantRole]
	if !hasKey {
		return nil, errors.Errorf("annotation '%s' is missing on client namespace '%s'", steward.AnnotationTenantRole, clientNamespace)
	}
	if value == "" {
		return nil, errors.Errorf("annotation '%s' on client namespace '%s' must not have an empty value", steward.AnnotationTenantRole, clientNamespace)
	}
	newConfig.tenantRoleName = k8s.RoleName(value)

	value, hasKey = annotations[steward.AnnotationTenantNamespaceSuffixLength]
	if hasKey {
		i, err := strconv.ParseInt(value, 10, 8)
		if err != nil {
			return nil, errors.Errorf(
				"annotation '%s' on client namespace '%s' has an invalid value: '%s':"+
					" should be a decimal integer in the range of [%d, %d]",
				steward.AnnotationTenantNamespaceSuffixLength, clientNamespace, value,
				math.MinInt8, math.MaxInt8)
		}
		newConfig.tenantNamespaceSuffixLength = i
	}
	return &newConfig, nil
}

func (c *clientConfigImpl) GetTenantNamespacePrefix() string {
	return c.tenantNamespacePrefix
}

func (c *clientConfigImpl) GetTenantNamespaceSuffixLength() uint8 {
	if c.tenantNamespaceSuffixLength < 0 {
		return tenantNamespaceSuffixLengthDefault
	}
	if c.tenantNamespaceSuffixLength > int64(tenantNamespaceSuffixLengthMax) {
		return tenantNamespaceSuffixLengthMax
	}
	return uint8(c.tenantNamespaceSuffixLength)
}

func (c *clientConfigImpl) GetTenantRoleName() k8s.RoleName {
	return c.tenantRoleName
}
