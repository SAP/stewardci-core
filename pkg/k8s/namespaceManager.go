package k8s

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strings"

	stuerrors "github.com/SAP/stewardci-core/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

//NamespaceManager manages namespaces
type NamespaceManager interface {
	Create(name string, annotations map[string]string) (string, error)
	Delete(name string) error
}

type namespaceManager struct {
	nsInterface  corev1.NamespaceInterface
	prefix       string
	randomLength int
}

//NewNamespaceManager creates NamespaceManager
func NewNamespaceManager(factory ClientFactory, prefix string, randomLength int) NamespaceManager {
	return &namespaceManager{
		nsInterface:  factory.CoreV1().Namespaces(),
		prefix:       prefix,
		randomLength: randomLength,
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
		log.Printf("Namespace creation failed %s", err)
		return "", err
	}
	meta := metav1.ObjectMeta{Name: name,
		Labels: map[string]string{
			labelPrefix: m.prefix,
			labelID:     nameCustomPart,
		}, Annotations: annotations}

	namespace := &v1.Namespace{ObjectMeta: meta}
	createdNamespace, err := m.nsInterface.Create(namespace)
	if err != nil {
		log.Printf("Namespace creation failed: %s", err)
		return "", err
	}
	log.Printf("Namespace '%s' created", createdNamespace.GetName())
	return createdNamespace.GetName(), nil
}

// Delete removes a namespace if existing
// returns nil error if deletion was successful or namespace did not exist before
func (m *namespaceManager) Delete(name string) error {
	if !strings.HasPrefix(name, m.prefix) {
		return fmt.Errorf("Cannot delete namespace '%s'. It does not start with prefix '%s'", name, m.prefix)
	}
	log.Printf("Deleting Namespace: '%s'", name)
	namespace, err := m.nsInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return stuerrors.Errorf(err, "Namespace deletion failed")
	}
	if namespace.GetLabels()[labelPrefix] == m.prefix {
		err = m.nsInterface.Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			} else {
				err = stuerrors.Errorf(err, "Namespace deletion failed")
			}
		} else {
			log.Printf("Namespace '%s' deleted", name)
		}
	} else {
		err = fmt.Errorf("Cannot delete namespace not owned by this steward instance: '%s'", name)
	}
	return err
}

//random creates a random hex value with length m.randomLength*2
func (m *namespaceManager) random() (string, error) {
	if m.randomLength < 0 {
		return "", errors.New("randomLength not configured in namespace manager")
	}
	if m.randomLength == 0 {
		return "", nil
	}
	b := make([]byte, m.randomLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func (m *namespaceManager) generateName(customPart string) (string, error) {
	var rand = ""
	var err error
	rand, err = m.random()
	if err != nil {
		return "", err
	}
	if rand != "" {
		rand = "-" + rand
	}
	if customPart != "" {
		customPart = "-" + customPart
	}
	name := fmt.Sprintf("%s%s%s", m.prefix, customPart, rand)
	return name, nil
}
