# Ruby script to rewrite the Jenkins-generated config for Deco to consume
# Expects one parameter - the file name of the json config to modify

require 'json'

# application config file that will be processed by Deco for parameter substitution
APPCONF = 'config/config.json'.freeze
# Deco-specific json to add
deco = { 'filters' => { APPCONF => {} } }

if ARGV.length.zero?
  puts 'No config file specified!'
  exit 1
end

CONFIG = ARGV.first

config = JSON.parse(File.read(CONFIG))
if config.is_a?(Array)
  puts "Rewriting #{CONFIG} in Deco format ..."
  config.each { |i| deco['filters'][APPCONF][i['key']] = i['value'] }
  File.open(CONFIG, 'w') { |f| f.puts JSON.pretty_generate(deco) }
else
  puts "Cannot process config file #{CONFIG}, aborting!"
  exit 1
end
