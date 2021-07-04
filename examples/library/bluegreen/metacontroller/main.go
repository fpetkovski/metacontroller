package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"metacontroller/pkg/apis/metacontroller/v1alpha1"
	"metacontroller/pkg/options"
	"metacontroller/pkg/server"
)

func main() {
	klog.InitFlags(nil)

	configuration := options.NewConfiguration()
	webhookUrl := "http://bluegreen-controller.metacontroller/sync"
	cc := v1alpha1.CompositeController{
		TypeMeta:   metav1.TypeMeta{
			Kind:       "CompositeController",
			APIVersion: "metacontroller.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "bluegreen-controller",
		},
		Spec:       v1alpha1.CompositeControllerSpec{
			ParentResource: v1alpha1.CompositeControllerParentResourceRule{
				ResourceRule: v1alpha1.ResourceRule{
					APIVersion: "ctl.enisoc.com/v1",
					Resource:   "bluegreendeployments",
				},
			},
			ChildResources: []v1alpha1.CompositeControllerChildResourceRule{
				{
					ResourceRule:   v1alpha1.ResourceRule{
						APIVersion: "v1",
						Resource:   "services",
					},
					UpdateStrategy: &v1alpha1.CompositeControllerChildUpdateStrategy{
						Method: v1alpha1.ChildUpdateInPlace,
					},
				},
				{
					ResourceRule:   v1alpha1.ResourceRule{
						APIVersion: "apps/v1",
						Resource:   "replicasets",
					},
					UpdateStrategy: &v1alpha1.CompositeControllerChildUpdateStrategy{
						Method: v1alpha1.ChildUpdateInPlace,
					},
				},
			},
			Hooks: &v1alpha1.CompositeControllerHooks{
				Sync: &v1alpha1.Hook{
					Webhook: &v1alpha1.Webhook{
						URL: &webhookUrl,
					},
				},
			},
		},
	}
	server.StartCompositeControllerServer(configuration, &cc)
}