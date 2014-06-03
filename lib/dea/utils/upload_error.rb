class UploadError < StandardError
  def initialize(msg, http, uri="(unknown)")
    uri = uri.to_s.gsub(/\/\/.*@/, '//[PRIVATE DATA HIDDEN]@')
    super("Error uploading: #{uri} (#{msg} status: #{http.response_header.status} - #{http.response})")
  end
end
