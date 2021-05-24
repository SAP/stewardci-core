package k8s

import (
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	klog "k8s.io/klog/v2"
	"strings"

	utils "github.com/SAP/stewardci-core/pkg/utils"
)

//NamespaceManager manages namespaces
type NamespaceManager interface {
	Create(name string, annotations map[string]string) (string, error)
	Delete(name string) error
	List(name string) ([]string, error)
}

type namespaceManager struct {
	nsInterface  corev1.NamespaceInterface
	prefix       string
	suffixLength uint8
}

// NewNamespaceManager creates a new NamespaceManager.
func NewNamespaceManager(factory ClientFactory, prefix string, suffixLength uint8) NamespaceManager {
	return &namespaceManager{
		nsInterface:  factory.CoreV1().Namespaces(),
		prefix:       prefix,
		suffixLength: suffixLength,
	}
}

const (
	labelPrefix = "prefix"
	labelID     = "id"
)

//Create creates a new namespace.
//    nameCustomPart	the namespace name will be <prefix>-<nameCustomPart>-<random>
//    annotations       annotations to create on the namespace
func (m *namespaceManager) Create(nameCustomPart string, annotations map[string]string) (string, error) {
	name, err := m.generateName(nameCustomPart)
	if err != nil {
		klog.V(2).Infof("Namespace creation failed %s", err)
		return "", err
	}
	meta := metav1.ObjectMeta{
		Name: name,
		Labels: map[string]string{
			labelPrefix: m.prefix,
			labelID:     nameCustomPart,
		},
		Annotations: annotations,
	}

	namespace := &v1.Namespace{ObjectMeta: meta}
	createdNamespace, err := m.nsInterface.Create(namespace)
	if err != nil {
		klog.V(2).Infof("Namespace creation failed: %s", err)
		return "", err
	}
	klog.V(2).Infof("Namespace '%s' created", createdNamespace.GetName())
	return createdNamespace.GetName(), nil
}

// Delete removes a namespace if existing
// returns nil error if deletion was successful or namespace did not exist before
func (m *namespaceManager) Delete(name string) error {
	if !strings.HasPrefix(name, m.prefix) {
		return errors.Errorf("refused to delete namespace '%s': name does not start with '%s'", name, m.prefix)
	}
	namespace, err := m.nsInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return errors.WithMessagef(err, "error getting namespace '%s'", name)
	}
	if namespace.GetLabels()[labelPrefix] != m.prefix {
		return errors.Errorf("refused to delete namespace '%s': not a Steward namespace (label mismatch)", name)
	}
	uid := namespace.GetObjectMeta().GetUID()
	err = m.nsInterface.Delete(name, &metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid},
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return errors.WithMessagef(err, "error deleting namespace '%s'", name)
	}
	klog.V(2).Infof("deleted namespace '%s'", name)
	return nil
}

func (m *namespaceManager) generateName(customPart string) (string, error) {
	parts := []string{}
	if m.prefix != "" {
		parts = append(parts, m.prefix)
	}
	if customPart != "" {
		parts = append(parts, customPart)
	}
	suffix, err := utils.RandomAlphaNumString(int64(m.suffixLength))
	if err != nil {
		return "", err
	}
	if suffix != "" {
		parts = append(parts, suffix)
	}
	return strings.Join(parts, "-"), nil
}

func (m *namespaceManager) List(nameCustomPart string) ([]string, error) {
	matchLables := map[string]string{
		labelPrefix: m.prefix,
		labelID:     nameCustomPart,
	}

	namespaces, err := m.nsInterface.List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v", labels.Set(matchLables).String()),
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		err = errors.WithMessagef(err, "error: failed to list namespaces with label selector %v", matchLables)
		return nil, err
	}

	var result []string
	for _, ns := range namespaces.Items {
		result = append(result, ns.Name)
	}

	return result, nil
}
