{ pkgs, app, name, version, ... }:
pkgs.dockerTools.buildImage {
  name = "docker.io/timescale/${name}";
  tag = version;

  copyToRoot = pkgs.buildEnv {
    name = "image-root";
    pathsToLink = [ "/bin" ];
    paths = [ app ];
  };
  created = "now";

  config = {
    WorkingDir = "${app}";
    Entrypoint = [ "./bin/${name}" ];
  };
}
