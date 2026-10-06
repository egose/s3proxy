"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[822],{

/***/ 6766
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_quickstart_md_807_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-quickstart-md-807.json
const site_docs_quickstart_md_807_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"quickstart","title":"Quickstart","description":"Choose the authenticated starter for binary-only onboarding, or the separate","source":"@site/docs/quickstart.md","sourceDirName":".","slug":"/quickstart","permalink":"/docs/quickstart","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":2,"frontMatter":{"sidebar_position":2},"sidebar":"docsSidebar","previous":{"title":"s3proxy","permalink":"/docs/intro"},"next":{"title":"Configuration","permalink":"/docs/configuration"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/quickstart.md


const frontMatter = {
	sidebar_position: 2
};
const contentTitle = 'Quickstart';

const assets = {

};



const toc = [{
  "value": "Authenticated Binary-Only Starter",
  "id": "authenticated-binary-only-starter",
  "level": 2
}, {
  "value": "Set Variables And Inspect Offline",
  "id": "set-variables-and-inspect-offline",
  "level": 3
}, {
  "value": "Prepare The Backend And Serve Natively",
  "id": "prepare-the-backend-and-serve-natively",
  "level": 3
}, {
  "value": "Starter Bounds And Timeouts",
  "id": "starter-bounds-and-timeouts",
  "level": 3
}, {
  "value": "Manual Auth-None Walkthrough",
  "id": "manual-auth-none-walkthrough",
  "level": 2
}, {
  "value": "Before You Start",
  "id": "before-you-start",
  "level": 2
}, {
  "value": "Install",
  "id": "install",
  "level": 2
}, {
  "value": "Minimal Config",
  "id": "minimal-config",
  "level": 2
}, {
  "value": "Export Secrets And Validate",
  "id": "export-secrets-and-validate",
  "level": 2
}, {
  "value": "Send Requests",
  "id": "send-requests",
  "level": 2
}, {
  "value": "Switch To SigV4 Auth",
  "id": "switch-to-sigv4-auth",
  "level": 2
}, {
  "value": "Next Steps",
  "id": "next-steps",
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
    p: "p",
    pre: "pre",
    strong: "strong",
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
        id: "quickstart",
        children: "Quickstart"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Choose the authenticated starter for binary-only onboarding, or the separate\nmanual auth-none walkthrough below for trusted local testing."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "authenticated-binary-only-starter",
      children: "Authenticated Binary-Only Starter"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["With ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " installed on your PATH (or substitute ", (0,jsx_runtime.jsx)(_components.code, {
        children: "./dist/s3proxy"
      }), " after\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "make build"
      }), "), print the version-owned HCL:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy print-example-config > config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Only your shell writes the file. The no-argument command prints deterministic HCL\nwith a trailing newline; it never loads configuration, evaluates environment\nvalues, generates credentials, binds a listener, or contacts a backend. Printing\nworks with every variable unset. Unsupported arguments/flags and output failures\nreturn nonzero; a failed output device may have accepted partial output."
    }), "\n", (0,jsx_runtime.jsx)(_components.h3, {
      id: "set-variables-and-inspect-offline",
      children: "Set Variables And Inspect Offline"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The following ", (0,jsx_runtime.jsx)(_components.strong, {
        children: "synthetic values are for offline inspection only"
      }), ". They do not\nprovision a backend or usable deployment credentials:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export S3PROXY_CLIENT_ACCESS_KEY=synthetic-client-access\nexport S3PROXY_CLIENT_SECRET_KEY=synthetic-client-secret\nexport S3PROXY_TARGET_PRIMARY_ENDPOINT=http://127.0.0.1:9000\nexport S3PROXY_TARGET_PRIMARY_ACCESS_KEY=synthetic-backend-access\nexport S3PROXY_TARGET_PRIMARY_SECRET_KEY=synthetic-backend-secret\n\ns3proxy validate --config ./config.hcl\ns3proxy routes --config ./config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["All five variables are required when loading the printed HCL. Missing variables\nfail validation safely; values stay literal even if they contain quotes or HCL\ntemplate syntax. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "validate"
      }), " prints ", (0,jsx_runtime.jsx)(_components.code, {
        children: "config is valid"
      }), ". Neither offline command\nrequires a running backend or an available listener port. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "routes"
      }), " reports exactly\none route:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["ordinal ", (0,jsx_runtime.jsx)(_components.code, {
          children: "1"
        }), ", label ", (0,jsx_runtime.jsx)(_components.code, {
          children: "images_rw"
        }), ", parser ", (0,jsx_runtime.jsx)(_components.code, {
          children: "images"
        }), " of kind ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_exact"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["operations, in order: ", (0,jsx_runtime.jsx)(_components.code, {
          children: "GetObject"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "HeadObject"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "PutObject"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "DeleteObject"
        }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
          children: "HeadBucket"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListObjectsV2"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["destination ", (0,jsx_runtime.jsx)(_components.code, {
          children: "primary"
        }), ", dispatch ", (0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        }), ", on_match ", (0,jsx_runtime.jsx)(_components.code, {
          children: "stop"
        }), ", read_preference ", (0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h3, {
      id: "prepare-the-backend-and-serve-natively",
      children: "Prepare The Backend And Serve Natively"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Before serving, replace the synthetic client key pair with your own private pair\nand replace the backend endpoint/key pair with credentials for your backend.\nKeep the two pairs separate: clients sign with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_CLIENT_*"
      }), ", while the proxy\nsigns upstream requests with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_TARGET_PRIMARY_*"
      }), ". The endpoint is the backend\nservice URL, not a bucket URL. Use HTTPS for a remote backend."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Provision ", (0,jsx_runtime.jsxs)(_components.strong, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "images-store"
        }), " on the backend in advance"]
      }), ", with permission for the\nbackend key to read/write/delete objects and list/head that bucket. The starter\nuses region ", (0,jsx_runtime.jsx)(_components.strong, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "us-east-1"
        })
      }), " and path-style addressing on both sides; edit the target\nregion if your backend requires another. The proxy does not create buckets."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The native listener is ", (0,jsx_runtime.jsx)(_components.strong, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "127.0.0.1:8080"
        })
      }), ". Only bucket ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images"
      }), " matches the single\nroute; its keys map unchanged into ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images-store"
      }), ". The sole client ", (0,jsx_runtime.jsx)(_components.code, {
        children: "local"
      }), " has an\nexplicit route grant, visible bucket ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images"
      }), ", the six routed operations above,\nand ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " permission. ListBuckets is served locally from virtual bucket\nmetadata, not forwarded through a catch-all route or used to discover backend buckets."]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy validate --config ./config.hcl\ns3proxy routes --config ./config.hcl\ns3proxy serve --config ./config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "In a separate client shell with the same private client variables exported:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export AWS_ACCESS_KEY_ID=\"$S3PROXY_CLIENT_ACCESS_KEY\"\nexport AWS_SECRET_ACCESS_KEY=\"$S3PROXY_CLIENT_SECRET_KEY\"\nexport AWS_DEFAULT_REGION=us-east-1\nunset AWS_SESSION_TOKEN AWS_SECURITY_TOKEN\naws --endpoint-url http://127.0.0.1:8080 s3api list-buckets\naws --endpoint-url http://127.0.0.1:8080 s3api head-bucket --bucket images\naws --endpoint-url http://127.0.0.1:8080 s3api put-object \\\n  --bucket images --key hello.txt --body ./hello.txt\naws --endpoint-url http://127.0.0.1:8080 s3api list-objects-v2 --bucket images\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Create ", (0,jsx_runtime.jsx)(_components.code, {
        children: "hello.txt"
      }), " before uploading. Use ordinary single-request uploads; multipart\nand AWS streaming payload formats are unsupported. See ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/request-examples",
        children: "request examples"
      }), "\nfor client compatibility and object read/delete commands."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h3, {
      id: "starter-bounds-and-timeouts",
      children: "Starter Bounds And Timeouts"
    }), "\n", (0,jsx_runtime.jsxs)(_components.table, {
      children: [(0,jsx_runtime.jsx)(_components.thead, {
        children: (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.th, {
            children: "Setting"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Printed value"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Scope"
          })]
        })
      }), (0,jsx_runtime.jsxs)(_components.tbody, {
        children: [(0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "replay_body_max_bytes"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: [(0,jsx_runtime.jsx)(_components.code, {
              children: "33554432"
            }), " (32 MiB)"]
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Per request when replay buffering is needed"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "replay_body_aggregate_max_bytes"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: [(0,jsx_runtime.jsx)(_components.code, {
              children: "268435456"
            }), " (256 MiB)"]
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Retained replay payloads across the process"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "timeouts.read_header"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "10s"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Request headers"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "timeouts.read"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "2m"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Entire inbound request, including upload body"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "timeouts.write"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "5m"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Response-write deadline, including handler/backend work"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "timeouts.idle"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "60s"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Keep-alive idle connection"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsxs)(_components.td, {
            children: ["target ", (0,jsx_runtime.jsx)(_components.code, {
              children: "timeout"
            })]
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "2m"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Complete upstream request and response stream"
          })]
        })]
      })]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["These are bounded starter settings for modest objects, not guarantees for every\nsize or network. Concrete signed payload hashes and unknown-length uploads can\nrequire replay even with one destination. Per-request overflow returns\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "413 EntityTooLarge"
      }), "; aggregate exhaustion returns ", (0,jsx_runtime.jsx)(_components.code, {
        children: "503 SlowDown"
      }), ". Known-length\nunsigned payloads may stream without replay, so the replay bound is not a global\nupload-size cap or a total process-memory cap. Size the bounds for memory and\nconcurrency, and align read/write/target and reverse-proxy timeouts for your slowest\nexpected uploads and downloads. The target timeout covers the entire download,\nnot just connection establishment."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For containers, follow ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/deployment#authenticated-starter-in-docker",
        children: "the Docker starter workflow"
      }), ":\noverride the serve-specific entrypoint for offline commands, change the container\nlistener to ", (0,jsx_runtime.jsx)(_components.code, {
        children: ":8080"
      }), ", and publish only a restricted host interface. SigV4 provides\nauthentication, not transport encryption; use the ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/deployment#reverse-proxying",
        children: "TLS/reverse-proxy guidance"
      }), "\nbefore exposing the service beyond local trusted access."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "manual-auth-none-walkthrough",
      children: "Manual Auth-None Walkthrough"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The remaining walkthrough builds from source and uses a hand-written config with\ninbound authentication disabled. It is distinct from the printed authenticated\nstarter and is intended for a trusted test environment."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "before-you-start",
      children: "Before You Start"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "You need:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "an S3-compatible backend such as MinIO"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "backend credentials with access to a bucket you want to expose"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "a local checkout of this repo"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Go 1.26 or newer and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "make"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "install",
      children: "Install"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Build from source:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make build\n./dist/s3proxy version\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "minimal-config",
      children: "Minimal Config"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Create ", (0,jsx_runtime.jsx)(_components.code, {
        children: "config.hcl"
      }), ":"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = false\n  }\n}\n\nauth \"main\" {\n  mode = \"none\"\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = env(\"S3PROXY_TARGET_PRIMARY_ENDPOINT\")\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"primary\"\n}\n\nparser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nroute \"images_rw\" {\n  parser          = \"images\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    bucket            = \"images-store\"\n  }\n}\n\nbucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["With that config, requests for ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/images/..."
      }), " are rewritten into the backend bucket ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images-store"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "export-secrets-and-validate",
      children: "Export Secrets And Validate"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If your config uses ", (0,jsx_runtime.jsx)(_components.code, {
        children: "env(\"...\")"
      }), ", export those variables before running the CLI:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export S3PROXY_TARGET_PRIMARY_ENDPOINT=http://127.0.0.1:9000\nexport S3PROXY_TARGET_PRIMARY_ACCESS_KEY=minioadmin\nexport S3PROXY_TARGET_PRIMARY_SECRET_KEY=minioadmin\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If you keep them in ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), ", load them first:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "set -a; . ./.env; set +a\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Validate the config:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy validate --config ./config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Start the proxy:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "s3proxy serve --config ./config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "From source, the equivalent command is:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "go run ./cmd/s3proxy serve --config ./config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "send-requests",
      children: "Send Requests"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For trusted local testing with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "mode = \"none\""
      }), ", the proxy skips inbound authentication. AWS CLI still needs credentials locally, so provide any placeholder values:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export AWS_ACCESS_KEY_ID=test\nexport AWS_SECRET_ACCESS_KEY=test\nexport AWS_DEFAULT_REGION=us-east-1\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Upload an object through the proxy:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url http://127.0.0.1:8080 s3api put-object \\\n  --bucket images \\\n  --key hello.txt \\\n  --body ./hello.txt\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "List objects through the same route:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url http://127.0.0.1:8080 s3api list-objects-v2 \\\n  --bucket images\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "List the virtual buckets exposed by the proxy:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url http://127.0.0.1:8080 s3api list-buckets\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "switch-to-sigv4-auth",
      children: "Switch To SigV4 Auth"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For anything outside a trusted environment, use ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sigv4_static"
      }), " so the proxy verifies the caller's S3 SigV4 signature."]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "auth \"main\" {\n  mode = \"sigv4_static\"\n\n  client \"local-dev\" {\n    access_key      = env(\"S3PROXY_CLIENT_ACCESS_KEY\")\n    secret_key      = env(\"S3PROXY_CLIENT_SECRET_KEY\")\n    allow_routes    = [\"route.images_rw\"]\n    visible_buckets = [\"images\"]\n  }\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Then point your S3 client or AWS CLI at the proxy with those client credentials:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export AWS_ACCESS_KEY_ID=$S3PROXY_CLIENT_ACCESS_KEY\nexport AWS_SECRET_ACCESS_KEY=$S3PROXY_CLIENT_SECRET_KEY\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The client credentials used to call the proxy are separate from the backend credentials used by ", (0,jsx_runtime.jsx)(_components.code, {
        children: "target \"s3\""
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "next-steps",
      children: "Next Steps"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Add more routes and rewrites in ", (0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/configuration",
          children: "Configuration"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Set up fan-out replication or failover in ", (0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/config-examples",
          children: "Config Examples"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Review exact behavior in ", (0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/api-reference",
          children: "API Reference"
        })]
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