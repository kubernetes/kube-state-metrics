set -e
set -x

# See: https://github.com/bazelbuild/rules_go#how-do-i-run-bazel-on-travis-ci
bazel --host_jvm_args=-Xmx500m \
    --host_jvm_args=-Xms500m \
    test \
    --spawn_strategy=standalone \
    --genrule_strategy=standalone \
    --test_strategy=standalone \
    --local_ram_resources=1536 \
    --noshow_progress \
    --verbose_failures \
    --test_output=errors \
    //:go_default_test
