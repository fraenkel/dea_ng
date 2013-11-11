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
	url *url.URL
	key string
}

func NewStagingMessage(data map[string]interface{}) StagingMessage {
	return StagingMessage{message: data}
}

func (msg StagingMessage) AsMap() map[string]interface{} {
	return msg.message
}

func (msg StagingMessage) app_id() string {
	return msg.message["app_id"].(string)
}

func (msg StagingMessage) task_id() string {
	return msg.message["task_id"].(string)
}

func (msg StagingMessage) properties() map[string]string {
	if props, ok := msg.message["properties"].(map[string]string); ok {
		return props
	}

	return nil
}

func (msg StagingMessage) buildpack_cache_upload_uri() *url.URL {
	return msg.staging_uri("buildpack_cache_upload_uri")
}

func (msg StagingMessage) buildpack_cache_download_uri() *url.URL {
	return msg.staging_uri("buildpack_cache_download_uri")
}

func (msg StagingMessage) upload_uri() *url.URL {
	return msg.staging_uri("upload_uri")
}

func (msg StagingMessage) download_uri() *url.URL {
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
	if adminBuildpacks, ok := msg.message["admin_buildpacks"].([]map[string]string); ok {
		buildpacks := make([]StagingBuildpack, 0, len(adminBuildpacks))
		for _, b := range adminBuildpacks {
			bpUrl, _ := url.Parse(b["url"])
			buildpacks = append(buildpacks, StagingBuildpack{bpUrl, b["key"]})
		}
		return buildpacks
	}
	return []StagingBuildpack{}
}

func (msg StagingMessage) staging_uri(key string) *url.URL {
	uri := msg.message[key].(string)
	if uri == "" {
		return nil
	}
	url, _ := url.Parse(uri)
	return url
}
