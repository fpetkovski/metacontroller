package server

import (
	"k8s.io/klog/v2"
	"metacontroller/pkg/apis/metacontroller/v1alpha1"
	mcclientset "metacontroller/pkg/client/generated/clientset/internalclientset"
	"metacontroller/pkg/controller/common"
	"metacontroller/pkg/controller/composite"
	"metacontroller/pkg/options"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func StartCompositeControllerServer(configuration options.Configuration, cc *v1alpha1.CompositeController)  {
	if configuration.RestConfig == nil {
		configuration.RestConfig = config.GetConfigOrDie()
	}

	mcClient, err := mcclientset.NewForConfig(configuration.RestConfig)
	if err != nil {
		klog.Fatal(err)
	}

	runtimeContext, err := common.NewControllerContext(configuration, mcClient)
	if err != nil {
		klog.Fatal(err)
	}
	runtimeContext.Start()
	runtimeContext.WaitForSync()

	ctrl, err := composite.NewParentController(
		runtimeContext.Resources,
		runtimeContext.DynClient,
		runtimeContext.DynInformers,
		runtimeContext.EventRecorder,
		runtimeContext.McClient,
		runtimeContext.McInformerFactory.Metacontroller().V1alpha1().ControllerRevisions().Lister(),
		cc,
		1,
	)
	if err != nil {
		klog.Fatal(err)
	}
	ctrl.Start()
	ctx := signals.SetupSignalHandler()
	<-ctx.Done()
	ctrl.Stop()
}
