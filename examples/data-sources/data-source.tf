data "timescale_products" "products" {
}

output "products_list" {
  value = data.timescale_products.products
}
