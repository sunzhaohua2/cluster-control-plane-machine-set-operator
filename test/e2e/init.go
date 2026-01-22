/*
Copyright 2023 Red Hat, Inc.

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

package e2e

import (
	"sync"

	. "github.com/onsi/gomega"

	"github.com/openshift/cluster-control-plane-machine-set-operator/test/e2e/framework"

	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var initOnce sync.Once

// InitializeTestFramework initializes the global test framework.
func InitializeTestFramework() {
	initOnce.Do(func() {
		if framework.GlobalFramework == nil {
			err := framework.InitFramework()
			Expect(err).NotTo(HaveOccurred(), "failed to initialize framework")
		}

		komega.SetClient(framework.GlobalFramework.GetClient())
		komega.SetContext(framework.GlobalFramework.GetContext())
	})
}
