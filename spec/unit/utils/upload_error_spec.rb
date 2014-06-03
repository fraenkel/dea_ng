require 'spec_helper'
require 'dea/utils/upload'
require 'uri'

describe UploadError do
  let (:http) { double(:http, response_header: double(:header, status: 200), response: "OK") }
  subject { UploadError.new("a message", http, uri) }

  context "no userinfo" do
    let (:uri) { "http://example.com/path?query=a" }

    describe "#to_s" do
      its(:to_s) { should == "Error uploading: http://example.com/path?query=a (a message status: 200 - OK)" }
    end
  end

  context "with userinfo" do
    let (:uri) { "http://user:pw@example.com/path?query=a" }

    describe "#to_s" do
      its(:to_s) { should == "Error uploading: http://[PRIVATE DATA HIDDEN]@example.com/path?query=a (a message status: 200 - OK)" }
    end
  end
end