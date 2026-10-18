package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

var errImageAsyncPluginSubmissionPending = errors.New("image async plugin submission or result is pending")

// submitOrResumeImageAsyncPluginTask keeps the plugin child as the only owner
// of billing. Once submission starts, a restart may only load the saved ID.
// An ambiguous upstream submission is never retried automatically.
func submitOrResumeImageAsyncPluginTask(c *gin.Context, parent *model.Task, pinned pluginruntime.PinnedEndpoint, deps pluginProtocolBridgeDeps, persist func() error) (*model.Task, error) {
	deps = deps.withDefaults()
	snapshot := parent.PrivateData.ImageAsync
	if snapshot.PluginTaskID == "" {
		return nil, errors.New("image async plugin child id is missing")
	}
	child, found, err := deps.getByTaskId(parent.UserId, snapshot.PluginTaskID)
	if err != nil {
		return nil, err
	}
	if found {
		if child.Platform != constant.TaskPlatform(snapshot.PluginKey) {
			return nil, errors.New("image async plugin child does not match its saved provider")
		}
		return child, nil
	}
	if snapshot.PluginSubmitted {
		return nil, errImageAsyncPluginSubmissionPending
	}
	if pinned.Plugin == nil || pinned.Plugin.Meta.Key != snapshot.PluginKey {
		return nil, errors.New("image async plugin is unavailable")
	}
	info, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		return nil, err
	}
	info.OriginModelName = c.GetString("resolved_task_model")
	info.Action = c.GetString("task_action")
	info.TaskRelayInfo.PublicTaskID = snapshot.PluginTaskID
	info.RequestId = fmt.Sprintf("image_async:%s", snapshot.PluginTaskID)
	c.Set(common.RequestIdKey, info.RequestId)
	c.Set("image_async_parent_task_id", parent.TaskID)
	snapshot.PluginSubmitted = true
	if persist == nil {
		return nil, errors.New("image async plugin submission barrier is unavailable")
	}
	if err := persist(); err != nil {
		snapshot.PluginSubmitted = false
		return nil, err
	}
	outcome, taskErr := deps.submit(c, info)
	if taskErr != nil {
		return nil, fmt.Errorf("%s: %s", taskErr.Code, taskErr.Message)
	}
	if outcome == nil || outcome.Task == nil || outcome.Task.TaskID != snapshot.PluginTaskID {
		return nil, errImageAsyncPluginSubmissionPending
	}
	return outcome.Task, nil
}

func runImageAsyncPluginRelay(task *model.Task, body []byte) (*imageAsyncRelayResult, error) {
	c, recorder, err := newImageAsyncReplayContext(task, body)
	if err != nil {
		return nil, err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			common.CleanupBodyStorage(c)
		}
	}()
	// Preparation executes the host-owned decoder and restores request context
	// needed by both fresh submission and the plugin's image renderer.
	engine := gin.New()
	replayKeys := c.Keys
	engine.POST(c.Request.URL.Path, func(replay *gin.Context) { replay.Keys = replayKeys; replay.Next() }, middleware.PinTaskPluginEndpoint(), middleware.PrepareTaskPluginEndpoint(), func(replay *gin.Context) {
		pinnedValue, _ := replay.Get(pluginruntime.ContextKeyPinnedEndpoint)
		pinned, ok := pinnedValue.(pluginruntime.PinnedEndpoint)
		if !ok || pinned.Plugin == nil || pinned.Plugin.Meta.Key != task.PrivateData.ImageAsync.PluginKey {
			err = errors.New("saved image async plugin is unavailable")
			return
		}
		deps := defaultPluginProtocolBridgeDeps()
		child, submitErr := submitOrResumeImageAsyncPluginTask(replay, task, pinned, deps, func() error {
			won, saveErr := task.ClaimImageAsyncPluginSubmission()
			if saveErr != nil {
				return saveErr
			}
			if !won {
				return errImageAsyncPluginSubmissionPending
			}
			return nil
		})
		if submitErr != nil {
			err = submitErr
			return
		}
		if child.Status != model.TaskStatusSuccess && child.Status != model.TaskStatusFailure {
			// The ordinary task poller owns completion and settlement; each worker
			// visit only observes its persisted state. No worker slot waits on it.
			err = errImageAsyncPluginSubmissionPending
			return
		}
		requestValue, _ := replay.Get(pluginruntime.ContextKeyProtocolRequest)
		protocolRequest, valid := requestValue.(pluginruntime.ProtocolRequestContext)
		if !valid {
			err = errors.New("image async plugin request context is missing")
			return
		}
		renderTaskPluginImageProtocolTask(replay, pinned, protocolRequest, child, deps)
		task.ChannelId = child.ChannelId
	})
	// HandleContext preserves the authenticated user/group context created
	// for replay, unlike issuing another HTTP request through the router.
	engine.HandleContext(c)
	if err != nil {
		return nil, err
	}
	if recorder.Code >= http.StatusBadRequest {
		return nil, fmt.Errorf("image plugin returned status %d: %s", recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
	if recorder.Body.Len() == 0 {
		return nil, errors.New("image plugin returned no result")
	}
	succeeded = true
	return &imageAsyncRelayResult{Body: recorder.Body.Bytes(), ChannelID: task.ChannelId, Context: c}, nil
}
