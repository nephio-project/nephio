/*
Copyright 2022-2023 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Environment struct {
	BaseName         string
	UniqueName       string
	clientConfig     *rest.Config
	Client           client.Client
	ClientSet        clientset.Interface
	manager          ctrl.Manager
	testEnv          *envtest.Environment
	cancelManagerCtx context.CancelFunc
	Namespace        *corev1.Namespace
	options          Options
}

type Options struct {
	UseManager            bool
	SkipNamespaceCreation bool
	RootPath              string
}

var RunID = uuid.NewUUID()

const charset = "abcdefghijklmnopqrstuvwxyz"

func RandomStringWithCharset(length int, charset string) string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandomAlphabaticalString(length int) string {
	return RandomStringWithCharset(length, charset)
}

func NewDefaultEnvironment(baseName string, options ...Options) *Environment {
	if len(options) > 1 {
		panic("provide atmost one Options as optional arg")
	}
	var option Options
	if len(options) > 0 {
		option = options[0]
	} else {
		option = Options{}
	}
	_, rootPath, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot get caller information")
	}
	rootPath = path.Join(rootPath, "..", "..")
	option.RootPath = rootPath
	return NewEnvironment(baseName, option)
}

func NewEnvironment(baseName string, options Options) *Environment {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

	var (
		rootPath string
		err      error
	)
	if len(options.RootPath) == 0 {
		panic("provide a root path to controller")
	}

	rootPath, err = filepath.Abs(options.RootPath)
	if err != nil {
		panic(err)
	}

	testEnv := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{filepath.Join(rootPath, "config", "crd", "bases")},
	}

	f := &Environment{
		BaseName: baseName,
		testEnv:  testEnv,
		options:  options,
	}

	return f
}

func (f *Environment) Start() {
	fmt.Println("bootstrapping test environment")
	cfg, err := f.testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).ToNot(BeNil())
	f.clientConfig = cfg

	f.ClientSet, err = clientset.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(f.ClientSet).ToNot(BeNil())

	if f.options.UseManager {
		f.manager = f.GetManager()
		f.Client = f.manager.GetClient()
	} else {
		f.Client, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(f.Client).ToNot(BeNil())

	if f.options.UseManager {
		go func() {
			var ctx context.Context
			ctx, f.cancelManagerCtx = context.WithCancel(ctrl.SetupSignalHandler())
			defer GinkgoRecover()
			Expect(f.manager.Start(ctx)).Should(Succeed())
		}()
	}
}

func (f *Environment) afterEach() {
	f.cancelManagerCtx()
}

// InitOnRunningSuite sets up ginkgo's BeforeEach & AfterEach.
// It must be called within running ginkgo suite (like Describe, Context etc)
func (f *Environment) InitOnRunningSuite() {
	BeforeEach(f.beforeEach)
	AfterEach(f.afterEach)
}

func (f *Environment) beforeEach() {
	if !f.options.SkipNamespaceCreation {
		By(fmt.Sprintf("Building a namespace api object, basename %s", f.BaseName))
		namespace, err := f.CreateNamespace(f.BaseName, map[string]string{
			"e2e-framework": f.BaseName,
		})
		Expect(err).NotTo(HaveOccurred())

		f.Namespace = namespace
		f.UniqueName = namespace.GetName()
	} else {
		f.UniqueName = fmt.Sprintf("%s-%s", f.BaseName, RandomAlphabaticalString(8))
	}
}

func (f Environment) CreateNamespace(baseName string, labels map[string]string) (*corev1.Namespace, error) {
	baseName = strings.ToLower(baseName)
	labels["e2e-run"] = string(RunID)
	name := fmt.Sprintf("%s-%s", baseName, RandomAlphabaticalString(8))
	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "",
			Labels:    labels,
		},
	}
	var got *corev1.Namespace
	var err error
	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		got, err = f.ClientSet.CoreV1().Namespaces().Create(context.TODO(), namespaceObj, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// regenerate on conflict
				Logf("Namespace name %q was already taken, generate a new name and retry", namespaceObj.Name)
				namespaceObj.Name = fmt.Sprintf("%v-%v", baseName, RandomAlphabaticalString(8))
			} else {
				Logf("Unexpected error while creating namespace: %v", err)
			}
		} else {
			break
		}
	}
	return got, err
}

func (f Environment) GetNamespace() string {
	if f.Namespace != nil {
		return f.Namespace.Name
	}
	return "default"
}

func (f *Environment) TeardownCluster() {
	err := f.testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
}

func (f Environment) GetRootPath() string {
	return f.options.RootPath
}
