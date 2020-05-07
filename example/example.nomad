job "example" {
  datacenters = ["dc1"]
  type        = "batch"

  group "example" {
    task "hello-world" {
      driver = "hello-world-example"

      config {
        greeting = "hello"
      }
    }
  }
}
