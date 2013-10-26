package starting

import "dea/env"

type StartMessage struct {
	message map[string]interface{}
	env.Message
}

func NewStartMessage(msg map[string]interface{}) StartMessage {
	return StartMessage{message: msg}
}

func (msg StartMessage) limits() map[string]uint64 {
	return msg.message["limits"].(map[string]uint64)
}

func (msg StartMessage) MemoryLimit() uint64 {
	return msg.limits()["mem"]
}

/*
class StartMessage
  def initialize(message)
    @message = message
  end

  def to_hash
    message
  end

  def index
    message["index"]
  end

  def droplet
    message["droplet"]
  end

  def version
    message["version"]
  end

  def name
    message["name"]
  end

  def uris
    (message["uris"] || []).map { |uri| URI(uri) }
  end

  def prod
    message["prod"]
  end

  def executable_uri
    URI(message["executableUri"]) if message["executableUri"]
  end

  def cc_partition
    message["cc_partition"]
  end


  def mem_limit
    limits["mem"]
  end

  def disk_limit
    limits["disk"]
  end

  def fds_limit
    limits["fds"]
  end

  def start_command
    message["start_command"]
  end

  def services
    message["services"] || []
  end

  def debug
    !!message["debug"]
  end

  def sha1
    message["sha1"]
  end

  def console
    !!message["console"]
  end

  def executable_file
    message["executableFile"]
  end

  def env
    message["env"] || []
  end

  def message
    @message || {}
  end
end
*/
