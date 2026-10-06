"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[588],{

/***/ 4236
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_deployment_md_a37_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-deployment-md-a37.json
const site_docs_deployment_md_a37_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"deployment","title":"Deployment","description":"This page covers practical ways to run s3proxy in local, containerized, and service-managed environments.","source":"@site/docs/deployment.md","sourceDirName":".","slug":"/deployment","permalink":"/docs/deployment","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":8,"frontMatter":{"sidebar_position":8},"sidebar":"docsSidebar","previous":{"title":"Operations","permalink":"/docs/operations"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/deployment.md


const frontMatter = {
	sidebar_position: 8
};
const contentTitle = 'Deployment';

const assets = {

};



const toc = [{
  "value": "Deployment Shape",
  "id": "deployment-shape",
  "level": 2
}, {
  "value": "Local Binary",
  "id": "local-binary",
  "level": 2
}, {
  "value": "Docker",
  "id": "docker",
  "level": 2
}, {
  "value": "Authenticated Starter In Docker",
  "id": "authenticated-starter-in-docker",
  "level": 3
}, {
  "value": "Docker Compose",
  "id": "docker-compose",
  "level": 2
}, {
  "value": "systemd",
  "id": "systemd",
  "level": 2
}, {
  "value": "Reverse Proxying",
  "id": "reverse-proxying",
  "level": 2
}, {
  "value": "Restarts And Rollouts",
  "id": "restarts-and-rollouts",
  "level": 2
}, {
  "value": "Production Recommendations",
  "id": "production-recommendations",
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
    strong: "strong",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "deployment",
        children: "Deployment"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This page covers practical ways to run ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " in local, containerized, and service-managed environments."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "deployment-shape",
      children: "Deployment Shape"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The project is built as:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["a single Go binary named ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["a container image built locally from the repo ", (0,jsx_runtime.jsx)(_components.code, {
          children: "Dockerfile"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["versioned container images published as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ghcr.io/egose/s3proxy:<version>"
        }), " when a release tag is published"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy exposes an HTTP S3-compatible endpoint. TLS termination is usually handled by an external load balancer or reverse proxy."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "local-binary",
      children: "Local Binary"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Build the binary:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make build\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Run it:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "./dist/s3proxy serve --config /etc/s3proxy/config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Validate config without starting the server:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "./dist/s3proxy validate --config /etc/s3proxy/config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "docker",
      children: "Docker"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Build the image:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "make docker-build\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Run it with a mounted config file:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "docker run --rm \\\n  -p 8080:8080 \\\n  -v ./config.hcl:/etc/s3proxy/config.hcl:ro \\\n  -e S3PROXY_CLIENT_CI_ACCESS_KEY=... \\\n  -e S3PROXY_CLIENT_CI_SECRET_KEY=... \\\n  -e S3PROXY_TARGET_PRIMARY_ACCESS_KEY=... \\\n  -e S3PROXY_TARGET_PRIMARY_SECRET_KEY=... \\\n  s3proxy\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If your config depends on more ", (0,jsx_runtime.jsx)(_components.code, {
        children: "env(\"...\")"
      }), " values, pass them in as environment variables or via an env file."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h3, {
      id: "authenticated-starter-in-docker",
      children: "Authenticated Starter In Docker"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The image entrypoint already includes ", (0,jsx_runtime.jsx)(_components.code, {
        children: "serve --config /etc/s3proxy/config.hcl"
      }), ".\nUse an explicit executable override for other CLI commands (the distroless image\nhas no shell). With a locally built ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " image:"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "docker run --rm --network none --entrypoint /usr/local/bin/s3proxy \\\n  s3proxy print-example-config > config.hcl\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The host shell writes ", (0,jsx_runtime.jsx)(_components.code, {
        children: "config.hcl"
      }), "; printing needs no variables or mounts. Edit\nits native ", (0,jsx_runtime.jsx)(_components.code, {
        children: "address = \"127.0.0.1:8080\""
      }), " to ", (0,jsx_runtime.jsxs)(_components.strong, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "address = \":8080\""
        }), " inside the container"]
      }), ".\nContainer loopback cannot receive traffic forwarded to published ports."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Prepare ", (0,jsx_runtime.jsx)(_components.code, {
        children: "starter.env"
      }), " with the five variables from the\n", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/quickstart#authenticated-binary-only-starter",
        children: "authenticated native quickstart"
      }), ":\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_CLIENT_ACCESS_KEY"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_CLIENT_SECRET_KEY"
      }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_TARGET_PRIMARY_ENDPOINT"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_TARGET_PRIMARY_ACCESS_KEY"
      }), ", and\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "S3PROXY_TARGET_PRIMARY_SECRET_KEY"
      }), ". Use private client keys and real backend\ncredentials, not the synthetic offline values. In a Docker env file use\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "NAME=value"
      }), " lines without shell ", (0,jsx_runtime.jsx)(_components.code, {
        children: "export"
      }), " statements. Protect that file and make\nthe mounted config readable by the image's nonroot user."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The endpoint must be reachable ", (0,jsx_runtime.jsx)(_components.strong, {
        children: "from the container"
      }), "; ", (0,jsx_runtime.jsx)(_components.code, {
        children: "127.0.0.1:9000"
      }), " there means\nthe container itself, not the host or another backend container. Use an appropriate\nbackend DNS name on your container network or a routable endpoint. Pre-create\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "images-store"
      }), ", and check the starter's ", (0,jsx_runtime.jsx)(_components.code, {
        children: "us-east-1"
      }), " region and timeout/replay bounds\nagainst your deployment."]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "docker run --rm --network none --entrypoint /usr/local/bin/s3proxy \\\n  --env-file ./starter.env -v \"$PWD/config.hcl:/etc/s3proxy/config.hcl:ro\" \\\n  s3proxy validate --config /etc/s3proxy/config.hcl\ndocker run --rm --network none --entrypoint /usr/local/bin/s3proxy \\\n  --env-file ./starter.env -v \"$PWD/config.hcl:/etc/s3proxy/config.hcl:ro\" \\\n  s3proxy routes --config /etc/s3proxy/config.hcl\ndocker run --rm -p 127.0.0.1:8080:8080 \\\n  --env-file ./starter.env -v \"$PWD/config.hcl:/etc/s3proxy/config.hcl:ro\" \\\n  s3proxy\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The last command uses the normal serve entrypoint and restricts publishing to host\nloopback. Add your backend container network as needed. For remote clients, apply\nnetwork access controls and ", (0,jsx_runtime.jsx)(_components.a, {
        href: "#reverse-proxying",
        children: "TLS termination"
      }), "; SigV4 does not\nencrypt HTTP traffic. These are deployment instructions, not a claim of live\nDocker/backend verification. Automated starter checks exercise native CLI output,\nvalidation, and topology offline."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "docker-compose",
      children: "Docker Compose"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Example ", (0,jsx_runtime.jsx)(_components.code, {
        children: "compose.yaml"
      }), ":"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-yaml",
        children: "services:\n  s3proxy:\n    image: s3proxy:latest\n    ports:\n      - '8080:8080'\n    env_file:\n      - .env\n    volumes:\n      - ./config.hcl:/etc/s3proxy/config.hcl:ro\n    restart: unless-stopped\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The example uses the image produced by ", (0,jsx_runtime.jsx)(_components.code, {
        children: "make docker-build"
      }), "; it does not assume a public ", (0,jsx_runtime.jsx)(_components.code, {
        children: "latest"
      }), " image."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Published GHCR images use semantic-version, major/minor, and major tags. The publishing workflow does not create a public ", (0,jsx_runtime.jsx)(_components.code, {
        children: "latest"
      }), " tag. Before publishing, it smoke-tests and scans the exact local image, then verifies the pushed image digest and attaches provenance and SBOM attestations."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "systemd",
      children: "systemd"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Example unit file:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-ini",
        children: "[Unit]\nDescription=s3proxy\nAfter=network-online.target\nWants=network-online.target\n\n[Service]\nType=simple\nUser=s3proxy\nGroup=s3proxy\nWorkingDirectory=/etc/s3proxy\nEnvironmentFile=/etc/s3proxy/s3proxy.env\nExecStart=/usr/local/bin/s3proxy serve --config /etc/s3proxy/config.hcl\nRestart=on-failure\nRestartSec=5\n\n[Install]\nWantedBy=multi-user.target\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Recommended layout:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["binary at ", (0,jsx_runtime.jsx)(_components.code, {
          children: "/usr/local/bin/s3proxy"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["config at ", (0,jsx_runtime.jsx)(_components.code, {
          children: "/etc/s3proxy/config.hcl"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["env file at ", (0,jsx_runtime.jsx)(_components.code, {
          children: "/etc/s3proxy/s3proxy.env"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "reverse-proxying",
      children: "Reverse Proxying"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " is often run behind an ingress, load balancer, or reverse proxy that handles:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "TLS termination"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "public DNS and certificates"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "network-level access control"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "request logging outside the application process"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If you rely on virtual-hosted addressing, make sure the outer layer preserves the bucket-bearing ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Host"
      }), " header shape the proxy expects."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "restarts-and-rollouts",
      children: "Restarts And Rollouts"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "There is no hot config reload in v1."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use an explicit restart when changing:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "auth clients"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "targets"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "routes"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "virtual buckets"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "listener settings"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Before rollout:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ol, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Run ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy validate --config ..."
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["Confirm every ", (0,jsx_runtime.jsx)(_components.code, {
          children: "env(\"...\")"
        }), " variable is present in the deployment environment."]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "Verify upstream connectivity and backend credentials."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "Exercise one read path and one write path through the proxy."
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["If using ", (0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"all\""
        }), " or ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        }), ", test those behaviors before production rollout."]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "validate"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "routes"
      }), " check listener TCP ", (0,jsx_runtime.jsx)(_components.code, {
        children: "host:port"
      }), " structure and the numeric\nport range offline. Missing port separators, URL-form addresses, and out-of-range\nnumeric ports now fail this gate before any success output or startup logging.\nWildcard hosts, hostnames, bracketed IPv6 with zones, named services, and zero or\nempty (ephemeral) ports remain supported. See the ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/configuration#listener-address-validation",
        children: "listener address contract"
      }), ".\nPassing validation does not establish hostname/service resolution, interface or\naddress ownership, bind permissions, or port availability in the deployment\nenvironment. Confirm those when starting the service; an occupied port or a valid\nbut unavailable local address intentionally passes offline inspection."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Configure load-balancer health checks as unsigned ", (0,jsx_runtime.jsx)(_components.strong, {
        children: "GET"
      }), " requests to the exact ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/readyz"
      }), " path on a localhost, IP, or base host, with no ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Authorization"
      }), " header and no query string (not even a bare ", (0,jsx_runtime.jsx)(_components.code, {
        children: "?"
      }), "). ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/healthz"
      }), " has the same request contract. Eligible probes return ", (0,jsx_runtime.jsx)(_components.code, {
        children: "200 OK"
      }), " with body ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ok"
      }), ", including on virtual-host-only listeners. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/readyz"
      }), " reports only that the process is serving requests; it does not probe target backends."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Configured virtual bucket hosts, signed/presigned requests, HEAD and other methods, encoded path spellings, and query-bearing requests use normal S3 handling instead of local probes. The exact unsigned query-free base-host GET shape is reserved for probes; unconfigured aliases cannot be recognized as virtual bucket hosts. Use the listener's configured host suffixes when choosing a health-check host. Restrict access at the network or reverse-proxy layer if needed. The distroless image does not include a shell or HTTP client for an in-container health command. See ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/api-reference#health-endpoints",
        children: "the API reference"
      }), " for the full boundary contract."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The process handles ", (0,jsx_runtime.jsx)(_components.code, {
        children: "SIGINT"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "SIGTERM"
      }), " with a 10-second graceful-shutdown window before active connections are forcibly closed. Configure service managers and orchestrators with a termination grace period longer than 10 seconds."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "production-recommendations",
      children: "Production Recommendations"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["use ", (0,jsx_runtime.jsx)(_components.code, {
          children: "sigv4_static"
        }), " unless the deployment is fully trusted"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "keep client credentials separate from backend target credentials"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "mount config files read-only"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["set explicit target ", (0,jsx_runtime.jsx)(_components.code, {
          children: "timeout"
        }), " values when using ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        }), ", accounting for the fact that the timeout covers the complete upstream response stream"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "monitor memory usage for fan-out writes, multi-route writes, concrete SigV4 payload hashes, and unknown-length request bodies"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "collect the structured JSON logs written to stdout; no metrics endpoint is exposed in v1"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "validate config before every deploy"
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