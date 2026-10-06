"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[81],{

/***/ 4852
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_operations_md_4de_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-operations-md-4de.json
const site_docs_operations_md_4de_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"operations","title":"Operations","description":"This page covers the commands and runtime behavior that matter most when working on or operating s3proxy.","source":"@site/docs/operations.md","sourceDirName":".","slug":"/operations","permalink":"/docs/operations","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":7,"frontMatter":{"sidebar_position":7},"sidebar":"docsSidebar","previous":{"title":"API Reference","permalink":"/docs/api-reference"},"next":{"title":"Deployment","permalink":"/docs/deployment"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/operations.md


const frontMatter = {
	sidebar_position: 7
};
const contentTitle = 'Operations';

const assets = {

};



const toc = [{
  "value": "Build And Run",
  "id": "build-and-run",
  "level": 2
}, {
  "value": "Offline Route Topology",
  "id": "offline-route-topology",
  "level": 2
}, {
  "value": "Environment Variables",
  "id": "environment-variables",
  "level": 2
}, {
  "value": "Docker",
  "id": "docker",
  "level": 2
}, {
  "value": "Unit Test And Validation Commands",
  "id": "unit-test-and-validation-commands",
  "level": 2
}, {
  "value": "Integration Tests",
  "id": "integration-tests",
  "level": 2
}, {
  "value": "Sandbox Commands",
  "id": "sandbox-commands",
  "level": 2
}, {
  "value": "Sandbox Isolation And Migration",
  "id": "sandbox-isolation-and-migration",
  "level": 3
}, {
  "value": "Runtime Behavior",
  "id": "runtime-behavior",
  "level": 2
}, {
  "value": "Logging And Diagnostics",
  "id": "logging-and-diagnostics",
  "level": 2
}, {
  "value": "Suggested Local Checklist",
  "id": "suggested-local-checklist",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    a: "a",
    code: "code",
    h1: "h1",
    h2: "h2",
    h3: "h3",
    header: "header",
    li: "li",
    ol: "ol",
    p: "p",
    pre: "pre",
    table: "table",
    tbody: "tbody",
    td: "td",
    th: "th",
    thead: "thead",
    tr: "tr",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "operations",
        children: "Operations"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This page covers the commands and runtime behavior that matter most when working on or operating ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "build-and-run",
      children: "Build And Run"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Common commands:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make build\npnpm release:build -- --version dev\npnpm release:verify -- --version dev --reproducibility\nmake run CONFIG=path/to/config.hcl\nmake validate CONFIG=path/to/config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The standard binary entrypoints are:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy serve --config /etc/s3proxy/config.hcl\ns3proxy validate --config /etc/s3proxy/config.hcl\ns3proxy routes --config /etc/s3proxy/config.hcl\ns3proxy print-example-config\ns3proxy version\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "serve"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "validate"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "routes"
      }), " also accept ", (0,jsx_runtime.jsx)(_components.code, {
        children: "-c"
      }), " as shorthand for ", (0,jsx_runtime.jsx)(_components.code, {
        children: "--config"
      }), ". These commands require a config path and do not accept positional arguments."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "print-example-config"
      }), " takes no arguments and prints a static authenticated starter\nwith literal environment references, without loading config or starting the runtime.\nUse ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy print-example-config > config.hcl"
      }), ", then follow the\n", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/quickstart#authenticated-binary-only-starter",
        children: "native starter workflow"
      }), " for the\nfive required variables, pre-existing backend bucket, and offline validation.\nFor containers, follow the ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/deployment#authenticated-starter-in-docker",
        children: "Docker starter instructions"
      }), "\nfor the executable entrypoint override and the required listener change to ", (0,jsx_runtime.jsx)(_components.code, {
        children: ":8080"
      }), "."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "make build"
      }), " produces ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dist/s3proxy"
      }), " for the host platform with CGO disabled."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "go-release.json"
      }), " defines the release target matrix and archive contract.\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "pnpm release:build"
      }), " creates deterministic ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy-<os>-<arch>.tar.gz"
      }), "\narchives and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "SHA256SUMS"
      }), "; ", (0,jsx_runtime.jsx)(_components.code, {
        children: "pnpm release:verify"
      }), " checks those artifacts and\ntheir exact version output. Add ", (0,jsx_runtime.jsx)(_components.code, {
        children: "--reproducibility"
      }), " to perform two independent\nrebuilds and compare archive hashes. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make build-all VERSION=<version>"
      }), " remains\na compatibility wrapper for the package build."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Keep ", (0,jsx_runtime.jsx)(_components.code, {
        children: "SHA256SUMS"
      }), " archive-only through package verification. The release\nworkflow appends the generated SBOM checksum only after verification and then\npublishes the archives, checksum manifest, and SBOM together."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "offline-route-topology",
      children: "Offline Route Topology"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Before a rollout, inspect ordered routing and replication with:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy routes --config /etc/s3proxy/config.hcl\n# Equivalent shorthand; suitable for saving or comparing topology:\ns3proxy routes -c /etc/s3proxy/config.hcl > routes.json\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The command fully loads and validates the configuration, including credentials,\nauth policies, bucket references, and rewrites. Load any environment variables\nfirst, just as for ", (0,jsx_runtime.jsx)(_components.code, {
        children: "validate"
      }), ". It prints only JSON on stdout, with no runtime logs,\napp construction, listener binding, or backend calls. Invalid config returns a\nnonzero exit status with existing safe diagnostics on stderr and no report on\nstdout. File/argument errors and output-write failures also return nonzero; an\noutput failure can leave a partial file, so check the exit status before using it."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The schema is an object with one ", (0,jsx_runtime.jsx)(_components.code, {
        children: "routes"
      }), " array. Each route contains:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.table, {
      children: [(0,jsx_runtime.jsx)(_components.thead, {
        children: (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.th, {
            children: "Field"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Meaning"
          })]
        })
      }), (0,jsx_runtime.jsxs)(_components.tbody, {
        children: [(0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ordinal"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "One-based position in config declaration order."
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "label"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Source-literal route declaration label."
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "parser"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Object with resolved declaration ", (0,jsx_runtime.jsx)(_components.code, {
              children: "label"
            }), " and ", (0,jsx_runtime.jsx)(_components.code, {
              children: "kind"
            }), ": ", (0,jsx_runtime.jsx)(_components.code, {
              children: "path_prefix"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "bucket_exact"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "bucket_regex"
            }), ", or ", (0,jsx_runtime.jsx)(_components.code, {
              children: "host_suffix"
            }), "."]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "operations"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Exact case-sensitive operation names, in configured order."
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "destinations"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Resolved target declaration labels, in configured order."
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "dispatch"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Required ", (0,jsx_runtime.jsx)(_components.code, {
              children: "first"
            }), " or ", (0,jsx_runtime.jsx)(_components.code, {
              children: "all"
            }), "."]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "on_match"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Required ", (0,jsx_runtime.jsx)(_components.code, {
              children: "stop"
            }), " or ", (0,jsx_runtime.jsx)(_components.code, {
              children: "continue"
            }), "."]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "read_preference"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: [(0,jsx_runtime.jsx)(_components.code, {
              children: "first"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "random"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "hash"
            }), ", or ", (0,jsx_runtime.jsx)(_components.code, {
              children: "ordered_failover"
            }), "; omitted/empty config values resolve to ", (0,jsx_runtime.jsx)(_components.code, {
              children: "first"
            }), "."]
          })]
        })]
      })]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For example, a route declared as ", (0,jsx_runtime.jsx)(_components.code, {
        children: "all"
      }), ", using parser ", (0,jsx_runtime.jsx)(_components.code, {
        children: "path_prefix"
      }), " named ", (0,jsx_runtime.jsx)(_components.code, {
        children: "all"
      }), ",\nwith ", (0,jsx_runtime.jsx)(_components.code, {
        children: "operations = [\"GetObject\"]"
      }), ", destination ", (0,jsx_runtime.jsx)(_components.code, {
        children: "primary"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch = \"first\""
      }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "on_match = \"stop\""
      }), ", and no read preference prints:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-json",
        children: "{\n  \"routes\": [\n    {\n      \"ordinal\": 1,\n      \"label\": \"all\",\n      \"parser\": {\n        \"label\": \"all\",\n        \"kind\": \"path_prefix\"\n      },\n      \"operations\": [\"GetObject\"],\n      \"destinations\": [\"primary\"],\n      \"dispatch\": \"first\",\n      \"on_match\": \"stop\",\n      \"read_preference\": \"first\"\n    }\n  ]\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["JSON uses two-space indentation and a trailing newline. Repeated invocations\nproduce identical bytes for the same evaluated topology. A valid config with no\nroutes prints ", (0,jsx_runtime.jsx)(_components.code, {
        children: "{\"routes\": []}"
      }), " (formatted across lines). Route, operation, and\ndestination arrays are never sorted; order matters. Bare and qualified references\nresolve to the same declaration labels, including references computed from ", (0,jsx_runtime.jsx)(_components.code, {
        children: "env()"
      }), "."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Operation filters never mean “all operations”: missing/empty lists, ", (0,jsx_runtime.jsx)(_components.code, {
        children: "*"
      }), ", duplicates,\nand unsupported names fail validation. Valid operation names are ", (0,jsx_runtime.jsx)(_components.code, {
        children: "GetObject"
      }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "HeadObject"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "PutObject"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "DeleteObject"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "HeadBucket"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ListObjectsV2"
      }), ", and\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), ". These are configured filters, not a claim that each request reaches\nthat route: ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " is handled locally, earlier matches can stop evaluation,\nand authorization can deny requests. Writes may fan out with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch = \"all\""
      }), ";\nreads still select one backend. The report includes the configured read preference,\nnot a simulated effective destination."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Only declaration labels and validated finite-vocabulary fields (plus ordinal) are\nexposed. Credentials, endpoints, regions, listener addresses, parser match strings\nand regexes, rewrite rules/templates, visible bucket values, environment names and\narbitrary values, and raw reference expressions are excluded. A qualified reference\nsuch as an environment-derived prefix followed by ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".primary"
      }), " prints only the\nresolved target declaration label ", (0,jsx_runtime.jsx)(_components.code, {
        children: "primary"
      }), ". Finite enum/operation values may be\nshown even when supplied by the environment. Keep secrets out of declaration\nlabels and config filenames: labels are public report metadata, and filenames and\nlabels are public diagnostic metadata."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This is configured topology, not request-routing simulation or a backend health\nor authorization check. It intentionally cannot explain regex captures or rewrite\nresults, and it does not prove credentials work against a backend."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "environment-variables",
      children: "Environment Variables"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The config loader evaluates ", (0,jsx_runtime.jsx)(_components.code, {
        children: "env(\"VAR\")"
      }), " as a native HCL function returning a\nliteral string. Environment contents are never parsed as HCL source or\ntemplates. Unset variables return an empty string and are then subject to\nordinary field validation. Parse/decode diagnostics use original file locations."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If you run locally with a ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), " file, load it before invoking the proxy:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "set -a; . ./.env; set +a\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "docker",
      children: "Docker"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Build and run with the repo helpers:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make docker-build\nmake docker-run CONFIG=path/to/config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The image mounts the config file and runs the same CLI entrypoint as the local binary."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "unit-test-and-validation-commands",
      children: "Unit Test And Validation Commands"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make vet\nmake test\nmake test-race\nmkdir -p dist\nmake cover\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "make cover"
      }), " writes ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dist/coverage.out"
      }), ", so ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dist/"
      }), " must already exist. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make build"
      }), " also creates it."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The standard local sanity check is:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make vet test\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "There is no separate typecheck target. A successful Go build is the typecheck."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "integration-tests",
      children: "Integration Tests"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The integration suite is build-tagged ", (0,jsx_runtime.jsx)(_components.code, {
        children: "integration"
      }), " and is skipped by ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make test"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Canonical one-shot flow:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "cp .env.example .env\nmake sandbox-integration-up\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The one-shot target tears down its proxy and sandbox after success or failure, including partial startup failures. The runner preserves the test/setup exit status; Make reports a failed recipe as exit 2 and prints the runner's status. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make sandbox-integration-down"
      }), " is also available for manual cleanup after an interrupted session."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Iterative flow:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make sandbox-up DAEMON=true\nmake build\nset -a; . ./.env; set +a\n./dist/s3proxy serve --config sandbox/integration-config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Keep the proxy running and use another terminal for repeated test runs:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "set -a; . ./.env; set +a\nmake test-integration\nmake test-integration-race\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "When finished, stop the proxy and tear down the sandbox:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make sandbox-down\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The sandbox stack exercises the proxy end to end against MinIO and SeaweedFS."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "sandbox-commands",
      children: "Sandbox Commands"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Useful sandbox helpers:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make sandbox-up\nmake sandbox-down\nmake sandbox-destroy\nmake sandbox-reset\nmake sandbox-logs\nmake sandbox-logs-follow\nmake sandbox-ps\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The sandbox compose file lives at ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sandbox/docker-compose.yml"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h3, {
      id: "sandbox-isolation-and-migration",
      children: "Sandbox Isolation And Migration"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["All Make helpers, integration discovery/cleanup, and CI validation use\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "scripts/sandbox-compose.sh"
      }), ". It passes ", (0,jsx_runtime.jsx)(_components.code, {
        children: "--project-name s3proxy-sandbox"
      }), " explicitly,\nprefers the ", (0,jsx_runtime.jsx)(_components.code, {
        children: "docker compose"
      }), " plugin, and falls back to standalone ", (0,jsx_runtime.jsx)(_components.code, {
        children: "docker-compose"
      }), ".\nFor direct Compose operations, use the same wrapper, for example:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "bash scripts/sandbox-compose.sh config --quiet\nbash scripts/sandbox-compose.sh ps -a\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["An alternate project must match ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy-[a-z0-9][a-z0-9_-]*"
      }), ". Set it in the shell\nenvironment or on every Make command line; use a name dedicated to this checkout:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export SANDBOX_PROJECT_NAME=s3proxy-my-worktree\nmake sandbox-up DAEMON=true\nmake sandbox-ps\nmake sandbox-down\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "SANDBOX_COMPOSE=/path/to/docker-compose"
      }), " optionally selects a single executable\n(no embedded arguments). Both overrides apply to direct script calls and Make\ntargets. Set them before invoking the runner, rather than inside ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), ".\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "COMPOSE_PROJECT_NAME"
      }), ", including a value loaded from ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), ", cannot replace the\nexplicit project. Integration cleanup retains the selection used at startup,\neven after ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), " is sourced for proxy/test credentials."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Containers now use Compose-generated project/service names instead of the old\nfixed ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy_*"
      }), " container names. Use ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ps"
      }), "/", (0,jsx_runtime.jsx)(_components.code, {
        children: "logs"
      }), " through the wrapper to discover\nthem. Networks and new named volumes are also project-scoped. Fixed host ports\n(including the proxy's 8082) still permit only one stack per host at a time;\na different project name does not isolate host ports or the local ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dist/"
      }), " PID/log\nfiles. MinIO, SeaweedFS, Azurite, fake-gcs-server, and s3-error remain included."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Historically the directory-derived identity was ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sandbox"
      }), ", shared with other\nrepositories. Startup's ", (0,jsx_runtime.jsx)(_components.code, {
        children: "--remove-orphans"
      }), " could delete their stopped containers.\nThe helpers now omit orphan removal, and destroy/reset no longer perform global\nimage pruning. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sandbox-down"
      }), " retains project volumes; ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sandbox-destroy"
      }), " and\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "sandbox-reset"
      }), " explicitly remove the selected stack's volumes and service images."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "For an existing installation:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ol, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Inspect existing resources without changing them:\n", (0,jsx_runtime.jsx)(_components.code, {
          children: "docker ps -a --filter label=com.docker.compose.project=sandbox"
        }), " and\n", (0,jsx_runtime.jsx)(_components.code, {
          children: "docker volume ls --filter label=com.docker.compose.project=sandbox"
        }), ".\nInspect individual container labels (", (0,jsx_runtime.jsx)(_components.code, {
          children: "com.docker.compose.service"
        }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
          children: "com.docker.compose.project.config_files"
        }), ", and\n", (0,jsx_runtime.jsx)(_components.code, {
          children: "com.docker.compose.project.working_dir"
        }), "), mounts, and published ports to\nestablish ownership. A shared project label or familiar name alone is not proof."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Leave unfamiliar containers, volumes, and services intact. Do not run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "down"
        }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
          children: "--remove-orphans"
        }), ", or volume/system prune against the shared ", (0,jsx_runtime.jsx)(_components.code, {
          children: "sandbox"
        }), " project.\nIf a legacy container occupies a required port, resolve it with its owner;\nstop/remove only individually verified, owned container IDs after preserving\nany needed data. A name/port conflict is not permission to remove a container."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Start the isolated project once its host ports are available. It creates fresh\nvolumes such as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy-sandbox_s3proxy_minio"
        }), "; old ", (0,jsx_runtime.jsx)(_components.code, {
          children: "sandbox_s3proxy_*"
        }), " volumes\nremain intact and are not adopted automatically. Back up and explicitly migrate\nneeded data with the owner's original configuration. Sandbox initialization\nrecreates test buckets, so it is not a data-preserving migration tool."]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "Containers already removed by a historical orphan cleanup cannot be restored\nfrom this repository alone. Obtain their original configuration from their\nowner before attempting recreation; retained volumes do not supply that config."
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "make test-sandbox"
      }), " exercises all lifecycle targets and integration cleanup using\nfake Docker/Compose commands in a temporary workspace. It requires no Docker\ndaemon and checks plugin/standalone selection, project overrides, unrelated\nresource preservation, failure status propagation, and proxy cleanup."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "runtime-behavior",
      children: "Runtime Behavior"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Important operational behaviors:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "only one listener is supported"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "config changes require a restart; there is no hot reload in v1"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["request bodies that need replay are buffered in memory up to ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_max_bytes"
        }), " per request and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_aggregate_max_bytes"
        }), " across the process"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "reads use one effective backend even when a route has multiple destinations"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["target ", (0,jsx_runtime.jsx)(_components.code, {
          children: "timeout"
        }), " is a deadline for the complete upstream exchange, including streaming the response body, and also affects failover timing for ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Replay buffering is used for fan-out writes, writes matched by multiple routes, inbound SigV4 requests with a concrete payload hash, and outbound requests whose body length is unknown. If the per-request replay limit is exceeded, the proxy returns ", (0,jsx_runtime.jsx)(_components.code, {
        children: "413 EntityTooLarge"
      }), " instead of attempting the upstream request. If the process aggregate replay budget is exhausted, the proxy returns ", (0,jsx_runtime.jsx)(_components.code, {
        children: "503 SlowDown"
      }), " immediately instead of blocking request goroutines."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "For memory sizing, the aggregate replay budget accounts for payload bytes, not\nprocess RSS. Unknown-length uploads coalesce fragmented reads into 32 KiB chunks;\nretained payload capacity equals the buffered byte count, including an exact-sized\nfinal chunk. Allow exactly one additional transient 32,768-byte scratch buffer per\nactive unknown-length buffering operation, plus chunk metadata, request bookkeeping,\nand Go allocator overhead. There is no extra retained final-chunk capacity allowance.\nMetadata scales with payload chunk count, not network read count. Unknown-length\nsources returning no bytes and no error 100 times consecutively fail instead of\nspinning; partial reservations are released on failure or cancellation."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "logging-and-diagnostics",
      children: "Logging And Diagnostics"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Use ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy validate --config ..."
      }), " before rollouts to catch configuration errors such as:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "invalid parser definitions"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "unknown target or route references"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "unsupported operation names"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "invalid auth mode or missing clients"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For integration troubleshooting, ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make sandbox-logs"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make sandbox-logs-follow"
      }), " are the fastest way to inspect backend behavior."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The proxy writes structured JSON logs to stdout. Each request receives an ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Request-Id"
      }), " response header; an inbound ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Request-Id"
      }), " is preserved, otherwise the proxy generates one. Request completion records include the method, status, response bytes, and duration. Dispatch records include route, operation, target, status, and sanitized errors when applicable. There is no runtime setting for log level, format, or destination in v1."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy does not expose a Prometheus or other metrics endpoint in v1. Collect stdout logs and process or container metrics externally."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "suggested-local-checklist",
      children: "Suggested Local Checklist"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ol, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Load environment variables used by ", (0,jsx_runtime.jsx)(_components.code, {
          children: "env(\"...\")"
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "make vet test"
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy validate --config ..."
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy routes --config ..."
        }), " and review route/destination order and modes."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Start the proxy and confirm ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        }), " and one object read/write path."]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "If using replication or failover, run the integration suite against the sandbox."
      }), "\n"]
    })]
  });
}
function MDXContent(props = {}) {
  const {wrapper: MDXLayout} = {
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return MDXLayout ? (0,jsx_runtime.jsx)(MDXLayout, {
    ...props,
    children: (0,jsx_runtime.jsx)(_createMdxContent, {
      ...props
    })
  }) : _createMdxContent(props);
}



/***/ },

/***/ 1982
(__unused_webpack___webpack_module__, __webpack_exports__, __webpack_require__) {

/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   R: () => (/* binding */ useMDXComponents),
/* harmony export */   x: () => (/* binding */ MDXProvider)
/* harmony export */ });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(489);
/**
 * @import {MDXComponents} from 'mdx/types.js'
 * @import {Component, ReactElement, ReactNode} from 'react'
 */

/**
 * @callback MergeComponents
 *   Custom merge function.
 * @param {Readonly<MDXComponents>} currentComponents
 *   Current components from the context.
 * @returns {MDXComponents}
 *   Additional components.
 *
 * @typedef Props
 *   Configuration for `MDXProvider`.
 * @property {ReactNode | null | undefined} [children]
 *   Children (optional).
 * @property {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @property {boolean | null | undefined} [disableParentContext=false]
 *   Turn off outer component context (default: `false`).
 */



/** @type {Readonly<MDXComponents>} */
const emptyComponents = {}

const MDXContext = react__WEBPACK_IMPORTED_MODULE_0__.createContext(emptyComponents)

/**
 * Get current components from the MDX Context.
 *
 * @param {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @returns {MDXComponents}
 *   Current components.
 */
function useMDXComponents(components) {
  const contextComponents = react__WEBPACK_IMPORTED_MODULE_0__.useContext(MDXContext)

  // Memoize to avoid unnecessary top-level context changes
  return react__WEBPACK_IMPORTED_MODULE_0__.useMemo(
    function () {
      // Custom merge via a function prop
      if (typeof components === 'function') {
        return components(contextComponents)
      }

      return {...contextComponents, ...components}
    },
    [contextComponents, components]
  )
}

/**
 * Provider for MDX context.
 *
 * @param {Readonly<Props>} properties
 *   Properties.
 * @returns {ReactElement}
 *   Element.
 * @satisfies {Component}
 */
function MDXProvider(properties) {
  /** @type {Readonly<MDXComponents>} */
  let allComponents

  if (properties.disableParentContext) {
    allComponents =
      typeof properties.components === 'function'
        ? properties.components(emptyComponents)
        : properties.components || emptyComponents
  } else {
    allComponents = useMDXComponents(properties.components)
  }

  return react__WEBPACK_IMPORTED_MODULE_0__.createElement(
    MDXContext.Provider,
    {value: allComponents},
    properties.children
  )
}


/***/ }

}]);