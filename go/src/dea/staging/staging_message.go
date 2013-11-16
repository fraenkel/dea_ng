package staging

import (
	"dea/starting"
	"net/url"
)

type StagingMessage struct {
	message   map[string]interface{}
	startData *starting.StartData
}

type StagingBuildpack struct {
	Url *url.URL
	Key string
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

func (msg StagingMessage) start_data() *starting.StartData {
	if msg.startData == nil {
		sdata := starting.NewStartData(msg.message["start_message"].(map[string]interface{}))
		msg.startData = &sdata
	}

	return msg.startData
}

func (msg StagingMessage) AdminBuildpacks() []StagingBuildpack {
	adminBuildpacks, ok := msg.message["admin_buildpacks"].([]map[string]interface{})
	if !ok {
		return []StagingBuildpack{}
	}

	buildpacks := make([]StagingBuildpack, 0, len(adminBuildpacks))
	for _, b := range adminBuildpacks {
		bpUrl, _ := url.Parse(b["url"].(string))
		buildpacks = append(buildpacks, StagingBuildpack{bpUrl, b["key"].(string)})
	}
	return buildpacks
}

func (msg StagingMessage) staging_uri(key string) *url.URL {
	uri := msg.message[key].(string)
	if uri == "" {
		return nil
	}
	url, _ := url.Parse(uri)
	return url
}
