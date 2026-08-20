output "namespace" {
  value = local.namespace
}

output "app_release" {
  value = module.app.release_name
}
