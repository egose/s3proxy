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
const site_docs_quickstart_md_807_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"quickstart","title":"Quickstart","description":"This quickstart runs s3proxy locally against one S3-compatible backend with inbound auth disabled.","source":"@site/docs/quickstart.md","sourceDirName":".","slug":"/quickstart","permalink":"/docs/quickstart","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":2,"frontMatter":{"sidebar_position":2},"sidebar":"docsSidebar","previous":{"title":"s3proxy","permalink":"/docs/intro"},"next":{"title":"Configuration","permalink":"/docs/configuration"}}');
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
        id: "quickstart",
        children: "Quickstart"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This quickstart runs ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " locally against one S3-compatible backend with inbound auth disabled."]
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
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["either an installed ", (0,jsx_runtime.jsx)(_components.code, {
          children: "s3proxy"
        }), " binary or a local checkout of this repo"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "install",
      children: "Install"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Install with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "asdf"
      }), ":"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "asdf plugin add s3proxy\n# or\nasdf plugin add s3proxy https://github.com/egose/s3proxy.git\n\nasdf install s3proxy latest\nasdf global s3proxy latest\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Or build from source:"
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
        children: "listener \"http\" \"public\" {\n  address = \":8080\"\n\n  addressing {\n    path_style     = true\n    virtual_hosted = false\n  }\n}\n\nauth \"main\" {\n  mode = \"none\"\n}\n\ncredential \"static\" \"primary\" {\n  access_key = env(\"S3PROXY_TARGET_PRIMARY_ACCESS_KEY\")\n  secret_key = env(\"S3PROXY_TARGET_PRIMARY_SECRET_KEY\")\n}\n\ntarget \"s3\" \"primary\" {\n  endpoint         = env(\"S3PROXY_TARGET_PRIMARY_ENDPOINT\")\n  region           = \"us-east-1\"\n  force_path_style = true\n  credentials      = \"primary\"\n}\n\nparser \"path_prefix\" \"images\" {\n  prefix = \"/images\"\n}\n\nroute \"images_rw\" {\n  parser          = \"images\"\n  operations      = [\"GetObject\", \"HeadObject\", \"PutObject\", \"DeleteObject\", \"ListObjectsV2\"]\n  destinations    = [\"primary\"]\n  dispatch        = \"first\"\n  on_match        = \"stop\"\n  read_preference = \"first\"\n\n  rewrite {\n    strip_path_prefix = \"/images\"\n    bucket            = \"images-store\"\n  }\n}\n\nbucket \"images\" {\n  visible_name = \"images\"\n  route        = \"images_rw\"\n}\n"
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