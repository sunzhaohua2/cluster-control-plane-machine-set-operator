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

package helpers

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	machinev1 "github.com/openshift/api/machine/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-control-plane-machine-set-operator/test/e2e/framework"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var (
	errUIDNotChanged = errors.New("UID has not changed")
)

// ExpectControlPlaneMachineSetToBeActive gets the control plane machine set and
// checks that it is active.
func ExpectControlPlaneMachineSetToBeActive() {
	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")
	k8sClient := framework.GlobalFramework.GetClient()

	cpms := &machinev1.ControlPlaneMachineSet{}
	Expect(k8sClient.Get(framework.GlobalFramework.GetContext(), framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms)).To(Succeed(), "control plane machine set should exist")

	Expect(cpms.Spec.State).To(Equal(machinev1.ControlPlaneMachineSetStateActive), "control plane machine set should be active")
}

// ExpectControlPlaneMachineSetToBeInactive gets the control plane machine set and
// checks that it is active.
func ExpectControlPlaneMachineSetToBeInactive() {
	By("Checking the control plane machine set is inactive")

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")
	k8sClient := framework.GlobalFramework.GetClient()

	cpms := &machinev1.ControlPlaneMachineSet{}
	Expect(k8sClient.Get(framework.GlobalFramework.GetContext(), framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms)).To(Succeed(), "control plane machine set should exist")

	Expect(cpms.Spec.State).To(Equal(machinev1.ControlPlaneMachineSetStateInactive), "control plane machine set should be inactive")
}

// ExpectControlPlaneMachineSetToBeInactiveOrNotFound gets the control plane machine set and
// checks that it is inactive or not found.
func ExpectControlPlaneMachineSetToBeInactiveOrNotFound() {
	By("Checking the control plane machine set is inactive or not found")

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")
	k8sClient := framework.GlobalFramework.GetClient()

	cpms := &machinev1.ControlPlaneMachineSet{}
	if err := k8sClient.Get(framework.GlobalFramework.GetContext(), framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms); err != nil {
		Expect(err).To(MatchError(ContainSubstring("not found")), "getting control plane machine set should not error")
	} else {
		Expect(cpms).To(HaveField("Spec.State", machinev1.ControlPlaneMachineSetStateInactive), "control plane machine set should be inactive or should not exist")
	}
}

// EnsureActiveControlPlaneMachineSet ensures that there is an active control plane machine set
// within the cluster. For fully supported clusters, this means waiting for the control plane machine set
// to be created and checking that it is active. For manually supported clusters, this means creating the
// control plane machine set, checking its status and then activating it.
func EnsureActiveControlPlaneMachineSet(gomegaArgs ...interface{}) {
	switch framework.GlobalFramework.GetPlatformSupportLevel() {
	case framework.Full:
		ensureActiveControlPlaneMachineSet(gomegaArgs...)
	case framework.Manual:
		ensureManualActiveControlPlaneMachineSet(gomegaArgs...)
	case framework.Unsupported:
		Fail(fmt.Sprintf("control plane machine set does not support platform %s", framework.GlobalFramework.GetPlatformType()))
	}
}

// EnsureInactiveControlPlaneMachineSet ensures that there is an inactive control plane machine set
// within the cluster.
func EnsureInactiveControlPlaneMachineSet(gomegaArgs ...interface{}) {
	switch framework.GlobalFramework.GetPlatformSupportLevel() {
	case framework.Full, framework.Manual:
		ensureInactiveControlPlaneMachineSet(gomegaArgs...)
	case framework.Unsupported:
		Fail(fmt.Sprintf("control plane machine set does not support platform %s", framework.GlobalFramework.GetPlatformType()))
	}
}

// ensureInactiveControlPlaneMachineSet checks that a CPMS exists and is inactive.
func ensureInactiveControlPlaneMachineSet(gomegaArgs ...interface{}) {
	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()
	ctx := framework.GlobalFramework.GetContext()

	By("Checking the control plane machine set exists")

	Eventually(komega.Get(cpms), gomegaArgs...).Should(Succeed(), "control plane machine set should exist")

	if cpms.Spec.State != machinev1.ControlPlaneMachineSetStateInactive {
		DeleteControlPlaneMachineSet(ctx, cpms)
	}

	By("Checking the control plane machine set is inactive")

	Eventually(komega.Object(cpms), gomegaArgs...).Should(HaveField("Spec.State", Equal(machinev1.ControlPlaneMachineSetStateInactive)), "control plane machine set should be inactive")
}

// ensureActiveControlPlaneMachineSet checks that a CPMS exists and then, if it is not active, activates it.
func ensureActiveControlPlaneMachineSet(gomegaArgs ...interface{}) {
	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()

	By("Checking the control plane machine set exists")

	Eventually(komega.Get(cpms), gomegaArgs...).Should(Succeed(), "control plane machine set should exist")

	if cpms.Spec.State != machinev1.ControlPlaneMachineSetStateActive {
		By("Activating the control plane machine set")

		Eventually(komega.Update(cpms, func() {
			cpms.Spec.State = machinev1.ControlPlaneMachineSetStateActive
		}), gomegaArgs...).Should(Succeed(), "control plane machine set should be able to be actived")
	}

	By("Checking the control plane machine set is active")

	Eventually(komega.Object(cpms), gomegaArgs...).Should(HaveField("Spec.State", Equal(machinev1.ControlPlaneMachineSetStateActive)), "control plane machine set should be active")
}

// ensureManualActiveControlPlaneMachineSet creates a CPMS if required and then activates it.
// If the CPMS already exists and is inactive, it will be activated.
func ensureManualActiveControlPlaneMachineSet(gomegaArgs ...interface{}) {
	k8sClient := framework.GlobalFramework.GetClient()
	ctx := framework.GlobalFramework.GetContext()

	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()
	if err := k8sClient.Get(ctx, framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms); err != nil && !apierrors.IsNotFound(err) {
		Fail(fmt.Sprintf("error when checking if a control plane machine set exists: %v", err))
	} else if err == nil {
		// The CPMS exists, so we just need to make sure it is active.
		ensureActiveControlPlaneMachineSet(gomegaArgs...)
		return
	}

	// A CPMS does not already exist, we must create one and then activate it.
	// TODO: Implement create functions for platforms that don't have generators yet.
	Fail("manual support for the control plane machine set not yet implemented")
}

// WaitForControlPlaneMachineSetDesiredReplicas waits for the control plane machine set to have the desired number of replicas.
// It first waits for the updated replicas to equal the desired number, and then waits for the final replica
// count to equal the desired number.
func WaitForControlPlaneMachineSetDesiredReplicas(ctx context.Context, cpms *machinev1.ControlPlaneMachineSet) bool {
	if ok := Expect(cpms.Spec.Replicas).ToNot(BeNil(), "replicas should always be set"); !ok {
		return false
	}

	desiredReplicas := *cpms.Spec.Replicas

	By("Waiting for the updated replicas to equal desired replicas")

	if ok := Eventually(komega.Object(cpms)).WithContext(ctx).Should(HaveField("Status.UpdatedReplicas", Equal(desiredReplicas)), "control plane machine set should have updated all replicas"); !ok {
		return false
	}

	By("Updated replicas is now equal to desired replicas")

	// Once the updated replicas equals the desired replicas, we need
	// to wait for the total replicas to go back to the desired replicas.
	// This will check the final machine gets removed before we end the test.
	By("Waiting for the replicas to equal desired replicas")

	if ok := Eventually(komega.Object(cpms)).WithContext(ctx).Should(HaveField("Status.Replicas", Equal(desiredReplicas)), "control plane machine set should have the desired number of replicas"); !ok {
		return false
	}

	By("Replicas is now equal to desired replicas")

	return true
}

// EnsureControlPlaneMachineSetUpdateStrategy ensures that the control plane machine set has the specified update strategy.
func EnsureControlPlaneMachineSetUpdateStrategy(strategy machinev1.ControlPlaneMachineSetStrategyType, gomegaArgs ...interface{}) machinev1.ControlPlaneMachineSetStrategyType {
	k8sClient := framework.GlobalFramework.GetClient()
	ctx := framework.GlobalFramework.GetContext()

	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()
	if err := k8sClient.Get(ctx, framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms); apierrors.IsNotFound(err) {
		Fail("control plane machine set does not exist")
	} else if err != nil {
		Fail(fmt.Sprintf("error when checking if a control plane machine set exists: %v", err))
	}

	originalStrategy := cpms.Spec.Strategy.Type
	if originalStrategy == strategy {
		return originalStrategy
	}

	By(fmt.Sprintf("Updating the control plane machine set strategy to %s", strategy))

	Eventually(komega.Update(cpms, func() {
		cpms.Spec.Strategy.Type = strategy
	}), gomegaArgs...).Should(Succeed(), "control plane machine set should be able to be updated")

	return originalStrategy
}

// EnsureControlPlaneMachineSetUpdated ensures the control plane machine set is up to date.
func EnsureControlPlaneMachineSetUpdated() {
	By("Checking the control plane machine set is up to date")

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")

	ctx := framework.GlobalFramework.GetContext()

	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()
	Eventually(komega.Get(cpms)).Should(Succeed(), "control plane machine set should exist")

	WaitForControlPlaneMachineSetDesiredReplicas(ctx, cpms)
}

// EnsureControlPlaneMachineSetDeleted ensures the control plane machine set
// is deleted and properly removed/recreated.
func EnsureControlPlaneMachineSetDeleted() {
	By("Ensuring the control plane machine set is deleted")

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")

	k8sClient := framework.GlobalFramework.GetClient()
	ctx := framework.GlobalFramework.GetContext()

	cpms := &machinev1.ControlPlaneMachineSet{}
	Expect(k8sClient.Get(framework.GlobalFramework.GetContext(), framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms)).
		To(Succeed(), "control plane machine set should exist")

	DeleteControlPlaneMachineSet(ctx, cpms)

	WaitForControlPlaneMachineSetRemovedOrRecreated(ctx, cpms.ObjectMeta.UID)
}

// DeleteControlPlaneMachineSet deletes the control plane machine set.
func DeleteControlPlaneMachineSet(ctx context.Context, cpms *machinev1.ControlPlaneMachineSet) {
	By("Deleting the control plane machine set")

	k8sClient := framework.GlobalFramework.GetClient()
	Expect(k8sClient.Delete(ctx, cpms)).To(Succeed(), "control plane machine set should have been deleted")
}

// WaitForControlPlaneMachineSetRemovedOrRecreated waits for the control plane machine set to be removed.
func WaitForControlPlaneMachineSetRemovedOrRecreated(ctx context.Context, oldCPMSUID types.UID) bool {
	By("Waiting for the deleted control plane machine set to be removed/recreated")

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")
	k8sClient := framework.GlobalFramework.GetClient()

	if ok := Eventually(func() error { //nolint:contextcheck
		newCPMS := &machinev1.ControlPlaneMachineSet{}
		if err := k8sClient.Get(framework.GlobalFramework.GetContext(), framework.GlobalFramework.ControlPlaneMachineSetKey(), newCPMS); err != nil {
			return fmt.Errorf("error getting control plane machine set: %w", err)
		}

		if newCPMS.GetUID() == oldCPMSUID {
			return errUIDNotChanged
		}

		return nil
	}).Should(SatisfyAny(
		BeNil(),
		MatchError("not found"),
	), "control plane machine set should be inactive or should not exist"); !ok {
		return false
	}

	By("Control plane machine set is now removed/recreated")

	return true
}

// ModifyControlPlaneMachineSetToTriggerRollout alters the provider spec of the
// control plane machine set. This modification must trigger
// the control plane machine set to update the machines based on
// the update strategy. An example would be changing the instance size.
func ModifyControlPlaneMachineSetToTriggerRollout(gomegaArgs ...interface{}) machinev1beta1.ProviderSpec {
	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()

	Eventually(komega.Get(cpms), gomegaArgs...).Should(Succeed(), "control plane machine set should exist")

	originalProviderSpec := cpms.Spec.Template.OpenShiftMachineV1Beta1Machine.Spec.ProviderSpec
	updatedProviderSpec := originalProviderSpec.DeepCopy()

	Expect(framework.GlobalFramework.ModifyProviderSpecToTriggerRollout(updatedProviderSpec.Value)).To(Succeed(), "provider spec should be updated")

	By("Modifying the control plane machine set provider spec")

	Eventually(komega.Update(cpms, func() {
		cpms.Spec.Template.OpenShiftMachineV1Beta1Machine.Spec.ProviderSpec = *updatedProviderSpec
	}), gomegaArgs...).Should(Succeed(), "control plane machine set should be able to be updated")

	return originalProviderSpec
}

// GetControlPlaneMachineSetUID gets the UID of the control plane machine set.
func GetControlPlaneMachineSetUID() types.UID {
	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")

	k8sClient := framework.GlobalFramework.GetClient()

	cpms := &machinev1.ControlPlaneMachineSet{}
	Expect(k8sClient.Get(framework.GlobalFramework.GetContext(), framework.GlobalFramework.ControlPlaneMachineSetKey(), cpms)).
		To(Succeed(), "control plane machine set should exist")

	return cpms.ObjectMeta.UID
}

// UpdateDefaultedValueFromControlPlaneMachineSetProviderConfig updates a defaulted field value from the Control Plane Machine Set's
// provider config to test defaulting on such value.
func UpdateDefaultedValueFromControlPlaneMachineSetProviderConfig(gomegaArgs ...interface{}) machinev1beta1.ProviderSpec {
	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")

	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()

	Eventually(komega.Get(cpms), gomegaArgs...).Should(Succeed(), "control plane machine set should exist")

	originalProviderSpec := cpms.Spec.Template.OpenShiftMachineV1Beta1Machine.Spec.ProviderSpec

	updatedProviderSpec := originalProviderSpec.DeepCopy()
	rawExtension, err := framework.GlobalFramework.UpdateDefaultedValueFromCPMS(updatedProviderSpec.Value)
	Expect(err).NotTo(HaveOccurred())

	updatedProviderSpec.Value = rawExtension

	By("Removing the defaulted field from the control plane machine set")

	Eventually(komega.Update(cpms, func() {
		cpms.Spec.Template.OpenShiftMachineV1Beta1Machine.Spec.ProviderSpec = *updatedProviderSpec
	}), gomegaArgs...).Should(Succeed(), "control plane machine set should be able to be updated")

	return originalProviderSpec
}

// UpdateControlPlaneMachineSetProviderSpec updates the provider spec of the control plane machine set to match the provider spec given.
func UpdateControlPlaneMachineSetProviderSpec(updatedProviderSpec machinev1beta1.ProviderSpec, gomegaArgs ...interface{}) {
	By("Updating the provider spec of the control plane machine set")

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")

	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()

	Eventually(komega.Get(cpms), gomegaArgs...).Should(Succeed(), "control plane machine set should exist")

	Eventually(komega.Update(cpms, func() {
		cpms.Spec.Template.OpenShiftMachineV1Beta1Machine.Spec.ProviderSpec = updatedProviderSpec
	}), gomegaArgs...).Should(Succeed(), "control plane machine should be able to be updated")
}

// UpdateControlPlaneMachineSetMachineNamePrefix updates the machine name prefix of the control plane machine set to match the machineNamePrefix given.
func UpdateControlPlaneMachineSetMachineNamePrefix(machineNamePrefix string, gomegaArgs ...interface{}) {
	if len(machineNamePrefix) > 0 {
		By(fmt.Sprintf("Updating the machine name prefix of the control plane machine set to %q", machineNamePrefix))
	} else {
		By("Un-setting the machine name prefix of the control plane machine set")
	}

	Expect(framework.GlobalFramework).ToNot(BeNil(), "test framework should not be nil")

	cpms := framework.GlobalFramework.NewEmptyControlPlaneMachineSet()

	Eventually(komega.Get(cpms), gomegaArgs...).Should(Succeed(), "control plane machine set should exist")

	Eventually(komega.Update(cpms, func() {
		cpms.Spec.MachineNamePrefix = machineNamePrefix
	}), gomegaArgs...).Should(Succeed(), "control plane machine should be able to be updated")
}
