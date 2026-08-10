"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[848],{

/***/ 3060
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_config_examples_md_0d0_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-config-examples-md-0d0.json
const site_docs_config_examples_md_0d0_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"config-examples","title":"Config Examples","description":"This page collects complete configuration examples for common s3proxy setups.","source":"@site/docs/config-examples.md","sourceDirName":".","slug":"/config-examples","permalink":"/docs/config-examples","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":5,"frontMatter":{"sidebar_position":5},"sidebar":"docsSidebar","previous":{"title":"Configuration","permalink":"/docs/configuration"},"next":{"title":"Routing and Rewrites","permalink":"/docs/providers-and-routing"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/config-examples.md


const frontMatter = {
	sidebar_position: 5
};
const contentTitle = 'Config Examples';

const assets = {

};



const toc = [{
  "value": "Local Development With One Backend",
  "id": "local-development-with-one-backend",
  "level": 2
}, {
  "value": "Static SigV4 Auth Plus One Route",
  "id": "static-sigv4-auth-plus-one-route",
  "level": 2
}, {
  "value": "Fan-Out Replication To Two Backends",
  "id": "fan-out-replication-to-two-backends",
  "level": 2
}, {
  "value": "Ordered Read Failover",
  "id": "ordered-read-failover",
  "level": 2
}, {
  "value": "Virtual-Hosted Bucket Matching",
  "id": "virtual-hosted-bucket-matching",
  "level": 2
}, {
  "value": "Tips",
  "id": "tips",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    code: "code",
    h1: "h1",
    h2: "h2",
    header: "header",
    li: "li",
    p: "p",
    pre: "pre",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "config-examples",
        children: "Config Examples"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This page collects complete configuration examples for common ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " setups."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use them as starting points, then adapt endpoints, bucket names, and secrets for your environment."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "local-development-with-one-backend",
      children: "Local Development With One Backend"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This is the smallest useful config. It exposes one path-based virtual bucket view and skips inbound auth."
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = false\n  }\n}\n\nauth \"main\" {\n  mode = \"none\"\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = env(\"S3PROXY_TARGET_PRIMARY_ENDPOINT\")\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"primary\"\n}\n\nparser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nroute \"images_rw\" {\n  parser          = \"images\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    strip_path_prefix = \"/images\"\n    bucket            = \"images-store\"\n  }\n}\n\nbucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use this when:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "you are testing locally"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "the proxy is behind another trusted boundary"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "one backend is enough"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "static-sigv4-auth-plus-one-route",
      children: "Static SigV4 Auth Plus One Route"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This adds inbound client authentication and limits the client to one route."
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = false\n  }\n}\n\nauth \"main\" {\n  mode = \"sigv4_static\"\n\n  client \"ci\" {\n    access_key      = env(\"S3PROXY_CLIENT_CI_ACCESS_KEY\")\n    secret_key      = env(\"S3PROXY_CLIENT_CI_SECRET_KEY\")\n    allow_routes    = [\"route.images_rw\"]\n    visible_buckets = [\"images\"]\n  }\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = \"http://127.0.0.1:9000\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"primary\"\n}\n\nparser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nroute \"images_rw\" {\n  parser          = \"images\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    strip_path_prefix = \"/images\"\n    bucket            = \"images-store\"\n  }\n}\n\nbucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use this when:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "you want the proxy to verify client SigV4 signatures"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "client-facing credentials must differ from backend credentials"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "you want a route-level allow-list"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "fan-out-replication-to-two-backends",
      children: "Fan-Out Replication To Two Backends"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This mirrors writes to a primary and replica backend."
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = false\n  }\n}\n\nauth \"main\" {\n  mode = \"sigv4_static\"\n\n  client \"writer\" {\n    access_key      = env(\"S3PROXY_CLIENT_WRITER_ACCESS_KEY\")\n    secret_key      = env(\"S3PROXY_CLIENT_WRITER_SECRET_KEY\")\n    allow_routes    = [\"route.replicate_rw\"]\n    visible_buckets = [\"replicate\"]\n  }\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ncredential \"static\" \"replica\" {\n  access_key = env(\"S3PROXY_TARGET_REPLICA_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_REPLICA_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = \"http://127.0.0.1:9000\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"primary\"\n}\n\ntarget \"s3\" \"replica\" {\n  endpoint         = \"http://127.0.0.1:8333\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"replica\"\n}\n\nparser \"path_prefix\" \"replicate_prefix\" {\n  prefix = \"/replicate\"\n}\n\nroute \"replicate_rw\" {\n  parser          = \"replicate_prefix\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\"]\n  destinations    = [\"primary\", \"replica\"]\n  dispatch        = \"all\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    strip_path_prefix = \"/replicate\"\n    bucket            = \"testbucket\"\n  }\n}\n\nbucket \"replicate\" {\n  visible_name = \"replicate\"\n  route        = \"replicate_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use this when:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "writes must land on every backend"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "reads can still prefer one backend"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["you have sized ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_max_bytes"
        }), " for the largest write you expect to replay and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_aggregate_max_bytes"
        }), " for expected concurrent replays"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "ordered-read-failover",
      children: "Ordered Read Failover"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This prefers one backend for reads and falls back only on transport failure, timeout, or upstream ", (0,jsx_runtime.jsx)(_components.code, {
        children: "5xx"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = false\n  }\n}\n\nauth \"main\" {\n  mode = \"sigv4_static\"\n\n  client \"reader\" {\n    access_key      = env(\"S3PROXY_CLIENT_READER_ACCESS_KEY\")\n    secret_key      = env(\"S3PROXY_CLIENT_READER_SECRET_KEY\")\n    allow_routes    = [\"route.failover_read\"]\n    visible_buckets = [\"failover\"]\n  }\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ncredential \"static\" \"replica\" {\n  access_key = env(\"S3PROXY_TARGET_REPLICA_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_REPLICA_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = \"http://10.0.0.10:9000\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  timeout          = \"250ms\"\n  credentials      = \"primary\"\n}\n\ntarget \"s3\" \"replica\" {\n  endpoint         = \"http://10.0.0.11:9000\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"replica\"\n}\n\nparser \"path_prefix\" \"failover_prefix\" {\n  prefix = \"/failover\"\n}\n\nroute \"failover_read\" {\n  parser          = \"failover_prefix\"\n  operations      = [\"GetObject\", \"HeadObject\"]\n  destinations    = [\"primary\", \"replica\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"ordered_failover\"\n\n  rewrite {\n    strip_path_prefix = \"/failover\"\n    bucket            = \"archive\"\n  }\n}\n\nbucket \"failover\" {\n  visible_name = \"failover\"\n  route        = \"failover_read\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use this when:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "you want a preferred backend and a warm backup"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "404"
        }), " from the primary should stay a ", (0,jsx_runtime.jsx)(_components.code, {
          children: "404"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "target timeout should bound failover latency"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "virtual-hosted-bucket-matching",
      children: "Virtual-Hosted Bucket Matching"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This enables bucket addressing through the host header."
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-hcl",
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = true\n    host_suffixes  = [\"s3proxy.example.com\"]\n  }\n}\n\nauth \"main\" {\n  mode = \"none\"\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = \"https://minio.internal\"\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"primary\"\n}\n\nparser \"bucket_exact\" \"assets\" {\n  bucket = \"assets\"\n}\n\nroute \"assets_rw\" {\n  parser          = \"assets\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\", \"HeadBucket\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    bucket = \"assets-prod\"\n  }\n}\n\nbucket \"assets\" {\n  visible_name = \"assets\"\n  route        = \"assets_rw\"\n}\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Use this when:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["clients address buckets through ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket.s3proxy.example.com"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "you want the visible bucket name to differ from the backend bucket"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "tips",
      children: "Tips"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "keep client credentials and backend credentials separate"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["use ", (0,jsx_runtime.jsx)(_components.code, {
          children: "visible_buckets"
        }), " to control what ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        }), " returns"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["prefer explicit target ", (0,jsx_runtime.jsx)(_components.code, {
          children: "timeout"
        }), " values for failover routes"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "avoid large fan-out writes unless you are comfortable buffering those bodies in memory"
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