#include "lmsdk-wrapper.h"

#include <sys/types.h>
#include <sys/wait.h>

int get_WIFEXITED(int status) {
    return WIFEXITED(status);
}

int get_WEXITSTATUS(int status) {
    return WEXITSTATUS(status);
}

int get_WIFSIGNALED(int status) {
    return WIFSIGNALED(status);
}

int get_WTERMSIG(int status) {
    return WTERMSIG(status);
}

int get_WCOREDUMP(int status) {
    return WCOREDUMP(status);
}