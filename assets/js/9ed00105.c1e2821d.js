"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[873],{

/***/ 2420
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_configuration_md_9ed_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-configuration-md-9ed.json
const site_docs_configuration_md_9ed_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"configuration","title":"Configuration","description":"s3proxy uses two-label HCL blocks. The core building blocks are:","source":"@site/docs/configuration.md","sourceDirName":".","slug":"/configuration","permalink":"/docs/configuration","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":3,"frontMatter":{"sidebar_position":3},"sidebar":"docsSidebar","previous":{"title":"Quickstart","permalink":"/docs/quickstart"},"next":{"title":"Config Examples","permalink":"/docs/config-examples"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/configuration.md


const frontMatter = {
	sidebar_position: 3
};
const contentTitle = 'Configuration';

const assets = {

};



const toc = [{
  "value": "Mental Model",
  "id": "mental-model",
  "level": 2
}, {
  "value": "Example",
  "id": "example",
  "level": 2
}, {
  "value": "Listener",
  "id": "listener",
  "level": 2
}, {
  "value": "Auth",
  "id": "auth",
  "level": 2
}, {
  "value": "Credentials",
  "id": "credentials",
  "level": 2
}, {
  "value": "Targets",
  "id": "targets",
  "level": 2
}, {
  "value": "Parsers",
  "id": "parsers",
  "level": 2
}, {
  "value": "Routes",
  "id": "routes",
  "level": 2
}, {
  "value": "Rewrites",
  "id": "rewrites",
  "level": 2
}, {
  "value": "Virtual Buckets",
  "id": "virtual-buckets",
  "level": 2
}, {
  "value": "Environment Variables",
  "id": "environment-variables",
  "level": 2
}, {
  "value": "Validation Rules",
  "id": "validation-rules",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    code: "code",
    h1: "h1",
    h2: "h2",
    header: "header",
    li: "li",
    ol: "ol",
    p: "p",
    pre: "pre",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "configuration",
        children: "Configuration"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " uses two-label HCL blocks. The core building blocks are:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "listener \"http\" \"public\""
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "auth \"main\""
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "credential \"static\" \"primary\""
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "target \"s3\" \"primary\""
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "parser \"path_prefix\" \"images\""
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "route \"images_rw\""
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket \"images\""
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "mental-model",
      children: "Mental Model"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Think about the config in seven layers:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ol, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "listener"
        }), " defines how the proxy accepts traffic."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "auth"
        }), " defines whether clients are authenticated and what they can see."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "credential"
        }), " defines backend credentials stored in config."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "target"
        }), " defines the S3-compatible backend endpoints."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "parser"
        }), " defines how requests are matched."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "route"
        }), " combines parser, operation filters, destinations, and rewrite rules."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "bucket"
        }), " defines the virtual buckets returned by ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        }), "."]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "example",
      children: "Example"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = true\n    host_suffixes  = [\"s3proxy.example.com\"]\n  }\n\n  timeouts {\n    read_header = \"10s\"\n    idle        = \"60s\"\n    write       = \"0s\"\n  }\n}\n\nauth \"main\" {\n  mode = \"sigv4_static\"\n\n  client \"ci\" {\n    access_key      = env(\"S3PROXY_CLIENT_CI_ACCESS_KEY\")\n    secret_key      = env(\"S3PROXY_CLIENT_CI_SECRET_KEY\")\n    allow_routes    = [\"route.images_rw\"]\n    visible_buckets = [\"images\"]\n  }\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = \"https://minio-a.internal\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  timeout          = \"5s\"\n  credentials      = \"primary\"\n}\n\nparser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nroute \"images_rw\" {\n  parser          = \"images\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    strip_path_prefix  = \"/images\"\n    prepend_key_prefix = \"assets/\"\n    bucket             = \"images-store\"\n  }\n}\n\nbucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "listener",
      children: "Listener"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The listener block configures the inbound HTTP server."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Important fields:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "address"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "max_header_bytes"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "addressing.path_style"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "addressing.virtual_hosted"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "addressing.host_suffixes"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "timeouts.read"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "timeouts.read_header"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "timeouts.idle"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "timeouts.write"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Notes:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["only one ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener"
        }), " block is supported"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["only ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener \"http\" ..."
        }), " is supported in v1"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["enabling ", (0,jsx_runtime.jsx)(_components.code, {
          children: "virtual_hosted"
        }), " requires at least one ", (0,jsx_runtime.jsx)(_components.code, {
          children: "host_suffix"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "auth",
      children: "Auth"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported inbound auth modes:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "none"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "sigv4_static"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "none"
      }), " skips inbound authentication and is only appropriate in trusted environments."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "sigv4_static"
      }), " validates inbound S3 SigV4 signatures against configured ", (0,jsx_runtime.jsx)(_components.code, {
        children: "client"
      }), " blocks."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Each client may define:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "access_key"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "secret_key"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "allow_routes"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "allow_ops"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "visible_buckets"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Example:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "auth \"main\" {\n  mode = \"sigv4_static\"\n\n  client \"admin\" {\n    access_key      = env(\"S3PROXY_CLIENT_ADMIN_ACCESS_KEY\")\n    secret_key      = env(\"S3PROXY_CLIENT_ADMIN_SECRET_KEY\")\n    allow_routes    = [\"*\"]\n    visible_buckets = [\"*\"]\n  }\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "credentials",
      children: "Credentials"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Backend credentials are defined separately from auth clients:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "credential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Only ", (0,jsx_runtime.jsx)(_components.code, {
        children: "credential \"static\" ..."
      }), " is supported in v1."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "targets",
      children: "Targets"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Targets define outbound S3-compatible backends:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "target \"s3\" \"primary\" {\n  endpoint         = \"https://minio-a.internal\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  timeout          = \"5s\"\n  credentials      = \"primary\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported fields:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "endpoint"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "region"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "force_path_style"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "timeout"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "credentials"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Only ", (0,jsx_runtime.jsx)(_components.code, {
        children: "target \"s3\" ..."
      }), " is supported in v1."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "parsers",
      children: "Parsers"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported parser kinds:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "path_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_exact"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_regex"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "host_suffix"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Examples:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "parser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nparser \"bucket_exact\" \"logs\" {\n  bucket = \"logs\"\n}\n\nparser \"bucket_regex\" \"tenant_logs\" {\n  pattern = \"^tenant-(?P<tenant>[a-z0-9-]+)-logs$\"\n}\n\nparser \"host_suffix\" \"public_hosts\" {\n  suffix = \"s3proxy.example.com\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Named regex captures from ", (0,jsx_runtime.jsx)(_components.code, {
        children: "bucket_regex"
      }), " are available to ", (0,jsx_runtime.jsx)(_components.code, {
        children: "key_template"
      }), " rewrites."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "routes",
      children: "Routes"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routes are evaluated in config order."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Each route combines:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "parser"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "operations"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "destinations"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "on_match"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "read_preference"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "rewrite"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Important route fields:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"first\""
        }), " sends the request to one destination"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"all\""
        }), " fans writes out to all destinations"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "on_match = \"stop\""
        }), " stops route evaluation after this match"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "on_match = \"continue\""
        }), " keeps collecting matches"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Supported ", (0,jsx_runtime.jsx)(_components.code, {
        children: "read_preference"
      }), " values:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "random"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "hash"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ordered_failover"
      }), ", failover happens only on transport errors, timeouts, and upstream ", (0,jsx_runtime.jsx)(_components.code, {
        children: "5xx"
      }), " responses."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "rewrites",
      children: "Rewrites"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported rewrite fields:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "strip_path_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "strip_key_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "prepend_key_prefix"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "key_template"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Example:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "rewrite {\n  strip_path_prefix  = \"/images\"\n  prepend_key_prefix = \"assets/\"\n  bucket             = \"images-store\"\n  key_template       = \"{{ .Captures.tenant }}/{{ .Key }}\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Template data uses the names ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Bucket"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Key"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Captures"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "virtual-buckets",
      children: "Virtual Buckets"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " returns buckets you define explicitly:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "bucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "visible_name"
      }), " is what the client sees. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "route"
      }), " decides how requests for that bucket are handled."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "environment-variables",
      children: "Environment Variables"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Use ", (0,jsx_runtime.jsx)(_components.code, {
        children: "env(\"VAR\")"
      }), " anywhere a string is allowed. The value is textually inlined before HCL parsing."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For local runs, load ", (0,jsx_runtime.jsx)(_components.code, {
        children: ".env"
      }), " before invoking the proxy if needed:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "set -a; . ./.env; set +a\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "validation-rules",
      children: "Validation Rules"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Startup fails on invalid configuration. Common checks include:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "missing listener or auth blocks"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "more than one listener or auth block"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "unsupported listener, credential, or target types"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "duplicate client, credential, target, or parser names"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["invalid parser config such as empty ", (0,jsx_runtime.jsx)(_components.code, {
          children: "prefix"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "pattern"
        }), ", or ", (0,jsx_runtime.jsx)(_components.code, {
          children: "suffix"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "routes that reference unknown parsers or destinations"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "invalid operation names or read preferences"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "sigv4_static"
        }), " auth without any clients"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "virtual buckets that reference unknown routes"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "CopyObject"
      }), " is recognized for classification but explicitly rejected as unsupported in v1."]
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