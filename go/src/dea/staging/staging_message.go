package staging

import (
	"dea"
	"dea/starting"
	"net/url"
)

type StagingMessage struct {
	message   map[string]interface{}
	startData *starting.StartData
}

func NewStagingMessage(data map[string]interface{}) StagingMessage {
	return StagingMessage{message: data}
}

func (msg StagingMessage) AsMap() map[string]interface{} {
	return msg.message
}

func (msg StagingMessage) App_id() string {
	return msg.message["app_id"].(string)
}

func (msg StagingMessage) Task_id() string {
	return msg.message["task_id"].(string)
}

func (msg StagingMessage) Properties() map[string]interface{} {
	if props, ok := msg.message["properties"].(map[string]interface{}); ok {
		return props
	}

	return nil
}

func (msg StagingMessage) Buildpack_cache_upload_uri() *url.URL {
	return msg.staging_uri("buildpack_cache_upload_uri")
}

func (msg StagingMessage) Buildpack_cache_download_uri() *url.URL {
	return msg.staging_uri("buildpack_cache_download_uri")
}

func (msg StagingMessage) Upload_uri() *url.URL {
	return msg.staging_uri("upload_uri")
}

func (msg StagingMessage) Download_uri() *url.URL {
	return msg.staging_uri("download_uri")
}

//func (msg StagingMessage) StartData() *starting.StartData {
func (msg StagingMessage) StartMessage() dea.StartMessage {
	if msg.startData == nil {
		sdata, err := starting.NewStartData(msg.message["start_message"].(map[string]interface{}))
		if err == nil {
			msg.startData = &sdata
		}
	}

	return msg.startData
}

func (msg StagingMessage) AdminBuildpacks() []dea.StagingBuildpack {
	adminBuildpacks, ok := msg.message["admin_buildpacks"].([]map[string]interface{})
	if !ok {
		return []dea.StagingBuildpack{}
	}

	buildpacks := make([]dea.StagingBuildpack, 0, len(adminBuildpacks))
	for _, b := range adminBuildpacks {
		bpUrl, _ := url.Parse(b["url"].(string))
		buildpacks = append(buildpacks, dea.StagingBuildpack{bpUrl, b["key"].(string)})
	}
	return buildpacks
}

func (msg StagingMessage) staging_uri(key string) *url.URL {
	v, exists := msg.message[key]
	if !exists {
		return nil
	}

	uri := v.(string)
	if uri == "" {
		return nil
	}
	url, _ := url.Parse(uri)
	return url
}
