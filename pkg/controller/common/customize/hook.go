package customize

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"metacontroller/pkg/apis/metacontroller/v1alpha1"
	"metacontroller/pkg/hooks"
)

var callCustomizeHook = hooks.Call

type CustomizableController interface {
	GetCustomizeHook() *v1alpha1.Hook
}

type CustomizeHookRequest struct {
	Controller CustomizableController     `json:"controller"`
	Parent     *unstructured.Unstructured `json:"parent"`
}

type CustomizeHookResponse struct {
	RelatedResourceRules []*v1alpha1.RelatedResourceRule `json:"relatedResources,omitempty"`
}

func CallCustomizeHook(cc CustomizableController, request *CustomizeHookRequest) (*CustomizeHookResponse, error) {
	var response CustomizeHookResponse

	hook := cc.GetCustomizeHook()
	// As the related hook is optional, return nothing
	if hook == nil {
		return &response, nil
	}

	if err := callCustomizeHook(hook, hooks.CustomizeHook, request, &response); err != nil {
		return nil, fmt.Errorf("related hook failed: %w", err)
	}

	return &response, nil
}
