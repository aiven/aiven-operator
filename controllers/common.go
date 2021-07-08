package controllers

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const conditionTypeRunning = "Running"
const conditionTypeInitialized = "Initialized"
const conditionTypeError = "Error"

func getInitializedCondition(reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeInitialized,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}

func getRunningCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeRunning,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func getErrorCondition(reason string, err error) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeError,
		Status:  metav1.ConditionUnknown,
		Reason:  reason,
		Message: err.Error(),
	}
}
