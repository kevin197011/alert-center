#!/usr/bin/env ruby
# frozen_string_literal: true
# Smoke test: login and hit API endpoints used by each page (no browser).

require 'net/http'
require 'json'
require 'uri'

BASE = ENV.fetch('BASE_URL', 'http://localhost:3000')
API = "#{BASE}/api/v1"

def request(method, path, body: nil, token: nil)
  uri = URI(path.start_with?('http') ? path : "#{API}#{path}")
  http = Net::HTTP.new(uri.host, uri.port)
  http.use_ssl = uri.scheme == 'https'
  http.open_timeout = 5
  http.read_timeout = 10
  req = case method.to_s.upcase
        when 'GET' then Net::HTTP::Get.new(uri)
        when 'POST' then Net::HTTP::Post.new(uri)
        else Net::HTTP::Get.new(uri)
        end
  req['Content-Type'] = 'application/json'
  req['Authorization'] = "Bearer #{token}" if token
  req.body = body.to_json if body
  res = http.request(req)
  { code: res.code.to_i, body: res.body }
rescue StandardError => e
  { code: -1, body: e.message }
end

def login
  res = request('POST', '/auth/login', body: { username: 'admin', password: 'admin123' })
  return nil unless res[:code] == 200

  data = JSON.parse(res[:body]) rescue {}
  data.dig('data', 'token')
end

# Endpoints that each page calls (GET list or main endpoint)
PAGE_ENDPOINTS = [
  ['/', 'GET', '/alert-history?page=1&page_size=10'],
  ['/', 'GET', '/alert-rules?page=1&page_size=10'],
  ['/rules', 'GET', '/alert-rules?page=1&page_size=10'],
  ['/channels', 'GET', '/channels?page=1&page_size=10'],
  ['/templates', 'GET', '/templates?page=1&page_size=10'],
  ['/history', 'GET', '/alert-history?page=1&page_size=10'],
  ['/users', 'GET', '/users?page=1&page_size=10'],
  ['/audit-logs', 'GET', '/audit-logs?page=1&page_size=10'],
  ['/data-sources', 'GET', '/data-sources?page=1&page_size=10'],
  ['/statistics', 'GET', '/statistics'],
  ['/silences', 'GET', '/silences?page=1&page_size=10'],
  ['/sla', 'GET', '/sla/configs'],
  ['/oncall', 'GET', '/oncall/schedules'],
  ['/correlation', 'GET', '/alert-history?page=1&page_size=100&status=firing'],
  ['/sla-breaches', 'GET', '/sla/breaches?page=1&page_size=10'],
  ['/oncall/report', 'GET', '/oncall/report'],
  ['/escalations', 'GET', '/escalations'],
  ['/tickets', 'GET', '/tickets?page=1&page_size=10'],
].freeze

def main
  puts "Smoke test against #{API}"
  token = login
  unless token
    puts "FAIL: Login failed (POST #{API}/auth/login)"
    exit 1
  end
  puts "OK: Login succeeded"

  failed = []
  PAGE_ENDPOINTS.each do |page, method, path|
    res = request(method, path, token: token)
    if res[:code] >= 200 && res[:code] < 300
      puts "OK: #{page} -> #{path} (#{res[:code]})"
    else
      puts "FAIL: #{page} -> #{path} (#{res[:code]}) #{res[:body][0..200]}"
      failed << [page, path, res[:code], res[:body][0..150]]
    end
  end

  if failed.any?
    puts "\nFailed endpoints:"
    failed.each { |p, path, code, msg| puts "  #{p} #{path} => #{code} #{msg}" }
    exit 1
  end
  puts "\nAll endpoints OK."
  exit 0
end

main
