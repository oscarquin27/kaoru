# Comprobación mínima y sin red del contrato: YAML válido, referencias que
# resuelven, operationId únicos y ningún componente sin uso.
# No reemplaza al lint de OpenAPI (reglas §A.18), que entra con el front en la fase 2.
require "yaml"

path = ARGV.fetch(0)
doc = YAML.load_file(path)
refs = []
ops = []
walk = lambda do |node|
  case node
  when Hash
    refs << node["$ref"] if node["$ref"].is_a?(String)
    ops << node["operationId"] if node["operationId"]
    node.each_value { |v| walk.(v) }
  when Array
    node.each { |v| walk.(v) }
  end
end
walk.(doc)

broken = refs.uniq.reject do |ref|
  ref.start_with?("#/") && ref[2..-1].split("/").reduce(doc) { |acc, key| acc.is_a?(Hash) ? acc[key] : nil }
end
repeated = ops.group_by(&:itself).select { |_, v| v.size > 1 }.keys
defined = %w[schemas parameters responses requestBodies].flat_map do |section|
  doc["components"][section].keys.map { |k| "#/components/#{section}/#{k}" }
end
unused = defined - refs.uniq

problems = { "referencias rotas" => broken, "operationId repetidos" => repeated, "componentes sin uso" => unused }
problems.each { |label, list| warn "FALLO: #{label}: #{list.join(', ')}" unless list.empty? }
exit 1 unless problems.values.all?(&:empty?)

puts "OpenAPI: OK (#{doc['paths'].size} rutas, #{ops.size} operaciones, #{doc['components']['schemas'].size} esquemas)"
