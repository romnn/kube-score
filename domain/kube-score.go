package domain

import (
	"io"

	autoscalingv1 "k8s.io/api/autoscaling/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Check struct {
	Name       string
	ID         string
	TargetType string
	Comment    string
	Optional   bool
}

type NamedReader interface {
	io.Reader
	Name() string
}

type FileLocation struct {
	Name string
	Skip bool
	Line int
}

type BothMeta struct {
	TypeMeta   metav1.TypeMeta
	ObjectMeta metav1.ObjectMeta
	FileLocationer
	// Annotations
}

type PodSpecer interface {
	FileLocationer
	// SkipInitContainers() bool
	GetTypeMeta() metav1.TypeMeta
	GetObjectMeta() metav1.ObjectMeta
	GetPodTemplateSpec() corev1.PodTemplateSpec
}

// type Annotations interface {
// 	Annotations() map[string]string
// }

type FileLocationer interface {
	FileLocation() FileLocation
}

type HpaTargeter interface {
	GetTypeMeta() metav1.TypeMeta
	GetObjectMeta() metav1.ObjectMeta
	MinReplicas() *int32
	HpaTarget() autoscalingv1.CrossVersionObjectReference
	FileLocationer
	// Annotations
}

type Ingress interface {
	GetTypeMeta() metav1.TypeMeta
	GetObjectMeta() metav1.ObjectMeta
	Rules() []networkingv1.IngressRule
	FileLocationer
	// Annotations
}

type Metas interface {
	Metas() []BothMeta
}

type Pod interface {
	Pod() corev1.Pod
	FileLocationer
	// Annotations
}

type Pods interface {
	Pods() []Pod
}

type PodSpeccers interface {
	PodSpeccers() []PodSpecer
}

type Service interface {
	Service() corev1.Service
	FileLocationer
	// Annotations
}

type Services interface {
	Services() []Service
}

type StatefulSet interface {
	StatefulSet() appsv1.StatefulSet
	FileLocationer
	// Annotations
}

type StatefulSets interface {
	StatefulSets() []StatefulSet
}

type Deployment interface {
	Deployment() appsv1.Deployment
	FileLocationer
	// Annotations
}

type Deployments interface {
	Deployments() []Deployment
}

type NetworkPolicy interface {
	NetworkPolicy() networkingv1.NetworkPolicy
	FileLocationer
	// Annotations
}

type NetworkPolicies interface {
	NetworkPolicies() []NetworkPolicy
}

type Ingresses interface {
	Ingresses() []Ingress
}

type Job interface {
	GetTypeMeta() metav1.TypeMeta
	GetObjectMeta() metav1.ObjectMeta
	GetPodTemplateSpec() corev1.PodTemplateSpec
	FileLocationer
	// Annotations
}

type Jobs interface {
	Jobs() []Job
}

type CronJob interface {
	GetTypeMeta() metav1.TypeMeta
	GetObjectMeta() metav1.ObjectMeta
	StartingDeadlineSeconds() *int64
	GetPodTemplateSpec() corev1.PodTemplateSpec
	FileLocationer
	// Annotations
}

type CronJobs interface {
	CronJobs() []CronJob
}

type PodDisruptionBudget interface {
	GetTypeMeta() metav1.TypeMeta
	GetObjectMeta() metav1.ObjectMeta
	Namespace() string
	Spec() policyv1.PodDisruptionBudgetSpec
	PodDisruptionBudgetSelector() *metav1.LabelSelector
	FileLocationer
	// Annotations
}

type PodDisruptionBudgets interface {
	PodDisruptionBudgets() []PodDisruptionBudget
}

type HorizontalPodAutoscalers interface {
	HorizontalPodAutoscalers() []HpaTargeter
}

type AllTypes interface {
	Metas
	Pods
	Jobs
	PodSpeccers
	Services
	StatefulSets
	Deployments
	NetworkPolicies
	Ingresses
	CronJobs
	PodDisruptionBudgets
	HorizontalPodAutoscalers
}
